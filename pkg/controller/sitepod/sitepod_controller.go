package sitepod

// Sitepod controller respond to additions/updates/deletions of sitepod resource type
// For new sitepods it provisions a linked (by label) persistent volume and deployment resource.
// For deletion performs similar action to kubectl reaper, change desired replicas to 0, waits
// and then removes the deployment and related replica sets.

import (
	"fmt"
	"reflect"
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
	sc := &SitepodController{*NewSimpleController("SitepodController", client, []Syncer{client.PVClaims(),
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
	if uid, hasUid := sc.Client.Sitepods().UIDOf(deleted); hasUid {
		sc.EnqueueDelete(uid)
	}
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
			return DependentConfigNotValid{fmt.Sprintf("No sitepod.io/pinned-host label specified for host local storage for pv %s", pv.GetName())}
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

	pvc := sitepod.Spec.VolumeClaims[0]

	if !smExists {
		vms := []k8s_api.VolumeMount{k8s_api.VolumeMount{
			MountPath: "/home",
			SubPath:   "home",
			Name:      "home-storage",
		}}

		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers,
			k8s_api.Container{
				VolumeMounts: vms,
				Name:         "sitepod-manager",
				Image:        "alpine:3.1",
				Command:      []string{"/usr/bin/tail", "-f", "/dev/null"},
			})

		homeStorageVolumeInPod := false
		for _, sv := range deployment.Spec.Template.Spec.Volumes {
			if sv.Name == "home-storage" {
				homeStorageVolumeInPod = true
				break
			}
		}

		// Find related PV for PVCLAIM
		if !homeStorageVolumeInPod {
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes,
				k8s_api.Volume{
					Name: "home-storage",
					VolumeSource: k8s_api.VolumeSource{
						PersistentVolumeClaim: &k8s_api.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc,
						},
					},
				})
		}
	}

	deployment.Spec.Template.GenerateName = "sitepod-pod-"
	deployment.Spec.Template.Labels = labels
	deployment.Labels = labels
	deployment = c.Deployments().UpdateOrAdd(deployment)

	if !sitepod.Status.StorageSetup {

		glog.Infof("Provisioning storage for sitepod %s", sitepodKey)
		pod, exists := c.Pods().MaybeSingleBySitepodKey(sitepodKey)
		if !exists {
			return ConditionsNotReady{"Still provisioning pod"}
		}

		readyExists, _ := From(pod.Status.Conditions).Where(func(s T) (bool, error) {
			return s.(k8s_api.PodCondition).Type == k8s_api.PodReady &&
				s.(k8s_api.PodCondition).Status == k8s_api.ConditionTrue, nil
		}).Any()

		if !readyExists {
			return ConditionsNotReady{"Pod not in ready state"}
		}

		podTasks := c.PodTasks().ByIndexByKey("sitepod", sitepodKey)

		//TODO figure out how to make this configurable
		cmd := []string{"/bin/mkdir", "-p", "/home"}

		podTaskExists := false
		for _, podTask := range podTasks {
			if reflect.DeepEqual(podTask.Spec.Command, cmd) && pod.Name == podTask.Spec.PodName {
				podTaskExists = true
				glog.Infof("Existing podtask for home dir creation for %s on %s found", key, sitepodKey)
			}
		}

		if podTaskExists {
			return ConditionsNotReady{"Pod task waiting completion"}
		}

		glog.Infof("Creating new pod task")
		podTask := c.PodTasks().NewEmpty()
		podTask.Labels = make(map[string]string)
		podTask.Labels["sitepod"] = sitepodKey
		podTask.Spec.Command = cmd
		podTask.Spec.PodName = pod.GetName()
		podTask.Spec.ContainerName = "sitepod-manager"
		podTask.Spec.Namespace = pod.GetNamespace()
		podTask.Spec.BehalfType = "Sitepod"
		podTask.Spec.BehalfOf = sitepod.Name
		podTask.Spec.BehalfCondition = "StorageReady"

		c.PodTasks().Add(podTask)
		glog.Infof("Created job to build storage for sitepod %s", sitepodKey)
	}

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

	dependencies := []struct {
		Getter  func(string) []interface{}
		Deleter func(interface{})
	}{
		{
			Getter:  c.PodTasks().BySitepodKeyFunc(),
			Deleter: c.PodTasks().DeleteFunc(),
		},
		{
			Getter:  c.SystemUsers().BySitepodKeyFunc(),
			Deleter: c.SystemUsers().DeleteFunc(),
		},
		{
			Getter:  c.AppComps().BySitepodKeyFunc(),
			Deleter: c.AppComps().DeleteFunc(),
		},
	}

	for _, dep := range dependencies {
		for _, v := range dep.Getter(key) {
			dep.Deleter(v)
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
