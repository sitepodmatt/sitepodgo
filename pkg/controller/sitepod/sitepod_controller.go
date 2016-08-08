package sitepod

// Sitepod controller respond to additions/updates/deletions of sitepod resource type
// For new sitepods it provisions a linked (by label) persistent volume and deployment resource.
// For deletion performs similar action to kubectl reaper, change desired replicas to 0, waits
// and then removes the deployment and related replica sets.

import (
	"fmt"
	"time"

	. "github.com/ahmetalpbalkan/go-linq"
	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	ext_api "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/controller/framework"
	//"sitepod.io/sitepod/pkg/api/v1"
	cc "sitepod.io/sitepod/pkg/client"
	. "sitepod.io/sitepod/pkg/controller/shared"
)

type SitepodController struct {
	SimpleController
}

func NewSitepodController(client *cc.Client) framework.ControllerInterface {

	glog.Infof("Creating sitepod controller")
	sc := &SitepodController{*NewSimpleController(client, []Syncer{client.PVClaims(),
		client.PVs(), client.Deployments()}, nil, nil)}
	sc.SyncFunc = sc.ProcessUpdate
	sc.DeleteFunc = sc.ProcessDelete
	client.Sitepods().AddInformerHandlers(framework.ResourceEventHandlerFuncs{
		AddFunc:    sc.QueueAdd,
		UpdateFunc: sc.QueueUpdate,
		DeleteFunc: sc.QueueDelete,
	})
	return sc
}

func (sc *SitepodController) Run(stopCh <-chan struct{}) {
	sc.WaitReady()
	//TODO how are we going to schedule the oprhan collector
	go sc.OrphanCollector(stopCh)
	sc.SimpleController.Run(stopCh)

}

func (sc *SitepodController) OrphanCollector(stopCh <-chan struct{}) {

	c := sc.Client
	// queue up - delete
	deleteThunks := []func(){}

	for _, deployment := range c.Deployments().List() {

		if sitepodLabel := deployment.Labels["sitepod"]; len(sitepodLabel) > 0 {
			_, exists := c.Sitepods().MaybeSingleByUID(sitepodLabel)
			if !exists {
				glog.Infof("Found orphan deployment %s", deployment.GetName())
				deleteThunks = append(deleteThunks, func() {
					// use the workqueue for error with retry are requeued
					// e.g. the multi stage cascade deletion of deployments
					sc.EnqueueDelete(sitepodLabel)
				})
			}
		}
	}

	//TODO - let clean up happen for very recently deleted items
	time.Sleep(15 * time.Second)

	for _, fn := range deleteThunks {
		func() {
			fn()
		}()

	}
}

func (sc *SitepodController) QueueAdd(item interface{}) {
	sc.EnqueueUpdate(sc.Client.Sitepods().KeyOf(item))
}

func (sc *SitepodController) QueueUpdate(old interface{}, cur interface{}) {
	if !sc.Client.Sitepods().DeepEqual(old, cur) {
		sc.EnqueueUpdate(sc.Client.Sitepods().KeyOf(cur))
	}
}

func (sc *SitepodController) QueueDelete(deleted interface{}) {
	sc.EnqueueDelete(sc.Client.Sitepods().KeyOf(deleted))
}

func (sc *SitepodController) ProcessUpdate(key string) error {

	c := sc.Client
	sitepod, exists := c.Sitepods().MaybeGetByKey(key)

	if !exists {
		glog.Infof("Sitepod %s not longer available. Presume this has since been deleted", key)
		return nil
	}

	sitepodKey := string(sitepod.UID)
	_ = sitepodKey

	if len(sitepod.Spec.VolumeClaims) == 0 {
		return DependentResourcesNotReady{fmt.Sprintf("Sitepod %s does not have any volume claims in spec", key)}
	}

	defaultPvc := sitepod.Spec.VolumeClaims[0]
	glog.Infof("Using pvc %s for sitepod %s", defaultPvc, key)

	pvClaim, exists := c.PVClaims().MaybeGetByKey(defaultPvc)

	if !exists {
		return DependentResourcesNotReady{fmt.Sprintf("PVC %s does not yet exist.", defaultPvc)}
	}

	if len(pvClaim.Spec.VolumeName) == 0 {
		return DependentResourcesNotReady{fmt.Sprintf("PVC %s exists but is not yet bound to a PV", defaultPvc)}
	}

	pv, exists := c.PVs().MaybeGetByKey(pvClaim.Spec.VolumeName)

	if !exists {
		return DependentResourcesNotReady{fmt.Sprintf("PV %s does not yet exist.", pvClaim.Spec.VolumeName)}
	}

	glog.Infof("Using pv %s for sitepod %s", pv.GetName(), key)

	isHostPath := pv.Spec.HostPath != nil
	var pinnedHost string
	if isHostPath {
		if pinnedHost = pv.Annotations["sitepod.io/pinned-host"]; len(pinnedHost) == 0 {
			return DependentConfigNotValid{fmt.Sprintf("No pinned host specified for host local storage for pv %s", pv.GetName())}
		}
	}

	deployment, exists := c.Deployments().MaybeSingleBySitepodKey(sitepodKey)

	if !exists {
		//TODO change NewForSitepod(sitepodKey)
		deployment = c.Deployments().NewEmpty()
		glog.Infof("Forging new deployment for sitepod %s", key)
	} else {
		glog.Infof("Using existing deployment %s for sitepod %s", deployment.GetName(), key)
	}

	labels := make(map[string]string)
	labels["sitepod"] = sitepodKey

	deployment.Spec.Replicas = 1
	///deployment.Spec.Selector = &unversioned.LabelSelector{MatchLabels: labels}

	if isHostPath {
		glog.Infof("Setting pinned host %s on deployment %s", pinnedHost, deployment.GetName())
		deployment.Spec.Template.Spec.NodeName = pinnedHost
	}

	smExists, _ := From(deployment.Spec.Template.Spec.Containers).Where(func(s T) (bool, error) {
		return (s.(k8s_api.Container).Name == "sitepod-manager"), nil
	}).Any()

	if !smExists {
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers,
			k8s_api.Container{
				Name:  "sitepod-manager",
				Image: "gcr.io/google_containers/pause:2.0",
			})
	}

	deployment.Spec.Template.GenerateName = "sitepod-pod-"
	deployment.Spec.Template.Labels = labels
	//TODO add labels don't replace
	deployment.Labels = labels

	deployment = c.Deployments().UpdateOrAdd(deployment)
	return nil
}

func (sc *SitepodController) ProcessDelete(key string) error {

	c := sc.Client

	deployments := c.Deployments().BySitepodKey(key)

	glog.Infof("Found %d deployments for sitepod  %s", len(deployments), key)

	for _, deployment := range deployments {
		err := sc.deleteDeployment(deployment)
		// TODO can we delete deployment concurrently
		if err != nil {
			return err
		}
	}

	return nil
}

// kubectl currently does all the heavy work for deployment cascade deletion
func (sc *SitepodController) deleteDeployment(deployment *ext_api.Deployment) error {

	//TODO stop this c = shit
	c := sc.Client

	glog.Infof("Winding down deployment %s", deployment.Name)

	if deployment.Spec.Replicas != 0 {

		glog.Infof("Setting replicas to 0 for %s", deployment.Name)
		deployment.Spec.Replicas = 0
		c.Deployments().Update(deployment)
		return ConditionsNotReady{"Set spec.replicas to zero"}
	}

	if deployment.Status.Replicas != 0 {
		return ConditionsNotReady{"Not zero status.replicas"}
	}

	c.Deployments().Delete(deployment)
	glog.Infof("Deleted deployment %s", deployment.Name)

	selector, _ := unversioned.LabelSelectorAsSelector(deployment.Spec.Selector)

	replicaSets := c.ReplicaSets().FetchList(selector)

	for _, replicaSet := range replicaSets {
		glog.Infof("Deleting %s of deployment %s",
			replicaSet.GetName(), deployment.GetName())

		err := c.ReplicaSets().TryDelete(replicaSet)
		if err != nil {
			if kerrors.IsNotFound(err) {
				glog.Infof("Already deleted %s of deployment %s",
					replicaSet.GetName(), deployment.GetName())
				continue
			} else {
				return err
			}
		} else {

			glog.Infof("Deleted %s of deployment %s",
				replicaSet.GetName(), deployment.GetName())
		}
	}

	return nil
}
