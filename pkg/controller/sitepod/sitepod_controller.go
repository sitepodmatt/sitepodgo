package sitepod

// Sitepod controller respond to additions/updates/deletions of sitepod resource type
// For new sitepods it provisions a linked (by label) persistent volume and deployment resource.
// For deletion performs similar action to kubectl reaper, change desired replicas to 0, waits
// and then removes the deployment and related replica sets.

import (
	"fmt"

	. "github.com/ahmetalpbalkan/go-linq"
	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
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
	client.Sitepods().AddInformerHandlers(framework.ResourceEventHandlerFuncs{
		AddFunc:    sc.AddFunc,
		UpdateFunc: sc.UpdateFunc,
		DeleteFunc: sc.DeleteFunc,
	})
	return sc
}

func (sc *SitepodController) AddFunc(item interface{}) {
	sc.EnqueueUpdate(sc.Client.Sitepods().KeyOf(item))
}

func (sc *SitepodController) UpdateFunc(old interface{}, cur interface{}) {
	if !sc.Client.Sitepods().DeepEqual(old, cur) {
		sc.EnqueueUpdate(sc.Client.Sitepods().KeyOf(cur))
	}
}

func (sc *SitepodController) DeleteFunc(deleted interface{}) {
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

	pv := c.PVs().GetByKey(pvClaim.Spec.VolumeName)
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

		glog.Infof("Deleting deployment %s", deployment.Name)

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

		selector, _ := unversioned.LabelSelectorAsSelector(deployment.Spec.Selector)

		replicaSets := c.ReplicaSets().FetchList(selector)

		for _, replicaSet := range replicaSets {
			glog.Infof("Deleting %s of deployment %s, of deleted sitepod %s",
				replicaSet.GetName(), deployment.GetName(), key)
			c.ReplicaSets().Delete(replicaSet)
		}

	}

	return nil
}
