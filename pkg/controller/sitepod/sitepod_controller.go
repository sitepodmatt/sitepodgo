package sitepod

// Sitepod controller respond to additions/updates/deletions of sitepod resource type
// For new sitepods it provisions a linked (by label) persistent volume and deployment resource.
// For deletion performs similar action to kubectl reaper, change desired replicas to 0, waits
// and then removes the deployment and related replica sets.

import (
	"os"
	"path"
	"time"

	"github.com/golang/glog"

	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/unversioned"
	ext_api "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/util/workqueue"
	"sitepod.io/sitepod/pkg/api/v1"
)

var (
	RetryDelay time.Duration = 200 * time.Millisecond
)

type SitepodController struct {
	sitepodInformer    framework.SharedIndexInformer
	pvInformer         framework.SharedIndexInformer
	deploymentInformer framework.SharedIndexInformer
	sitepodUpdater     v1.UpdaterFunc
	rcUpdater          v1.UpdaterFunc
	pvUpdater          v1.UpdaterFunc
	deploymentUpdater  v1.UpdaterFunc
	deploymentDeleter  v1.DeleterFunc
	rsFilter           v1.ListLabelFunc
	rsDeleter          v1.DeleterFunc
	queue              workqueue.DelayingInterface
}

func NewSitepodController(sitepodInformer framework.SharedIndexInformer,
	pvInformer framework.SharedIndexInformer,
	deploymentInformer framework.SharedIndexInformer,
	sitepodUpdater v1.UpdaterFunc,
	rcUpdater v1.UpdaterFunc,
	pvUpdater v1.UpdaterFunc,
	deploymentUpdater v1.UpdaterFunc,
	deploymentDeleter v1.DeleterFunc,
	rsFilter v1.ListLabelFunc,
	rsDeleter v1.DeleterFunc) framework.ControllerInterface {

	c := &SitepodController{sitepodInformer,
		pvInformer,
		deploymentInformer,
		sitepodUpdater,
		rcUpdater,
		pvUpdater,
		deploymentUpdater,
		deploymentDeleter,
		rsFilter,
		rsDeleter,
		workqueue.NewDelayingQueue(),
	}

	sitepodInformer.AddEventHandler(framework.ResourceEventHandlerFuncs{
		AddFunc:    c.queueAddSitepod,
		UpdateFunc: c.queueUpdateSitepod,
		DeleteFunc: c.queueDeleteSitepod})

	return c
}

type processSitepodRequest struct {
	id string
}

type deleteSitepodRequest struct {
	id string
}

var workQueueKeyFunc func(interface{}) (string, error) = uidKeyFunc

func uidKeyFunc(obj interface{}) (string, error) {
	sitepod := obj.(*v1.Sitepod)
	return string(sitepod.UID), nil
}

func (c *SitepodController) queueAddSitepod(obj interface{}) {
	sitepod := obj.(*v1.Sitepod)
	key, _ := workQueueKeyFunc(sitepod)
	c.queue.Add(processSitepodRequest{key})
}

func (c *SitepodController) queueUpdateSitepod(old interface{}, cur interface{}) {
	if k8s_api.Semantic.DeepEqual(old, cur) {
		return
	}
	sitepod := cur.(*v1.Sitepod)
	key, _ := workQueueKeyFunc(sitepod)
	c.queue.Add(processSitepodRequest{key})
}

func (c *SitepodController) queueDeleteSitepod(obj interface{}) {
	key, _ := workQueueKeyFunc(obj)
	c.queue.Add(deleteSitepodRequest{key})
}

func (c *SitepodController) Run(stopCh <-chan struct{}) {
	go c.worker()
	<-stopCh
	c.queue.ShutDown()
}

func (c *SitepodController) worker() {

	for !c.IsReady() {
		glog.Info("Waiting for dependencies to be ready")
		time.Sleep(RetryDelay)
	}

	for {
		func() {
			item, quit := c.queue.Get()
			if quit {
				return
			}
			defer c.queue.Done(item)

			if key, ok := item.(processSitepodRequest); ok {
				c.syncSitepod(key.id)
			}

			if key, ok := item.(deleteSitepodRequest); ok {
				c.deleteSitepod(key.id)
			}

		}()
	}
}

func (c *SitepodController) IsReady() bool {
	return (c.sitepodInformer.HasSynced() && c.deploymentInformer.HasSynced() && c.pvInformer.HasSynced())
}

func (c *SitepodController) syncSitepod(key string) {

	sitepodObjs, err := c.sitepodInformer.GetIndexer().ByIndex("uid", key)

	if err != nil {
		glog.Errorf("Unable to get sitepod %s: %+v", key, err)
		return
	}

	if len(sitepodObjs) == 0 {
		glog.Infof("Presuming sitepod %s has been deleted", key)
		return
	}

	sitepod := sitepodObjs[0].(*v1.Sitepod)
	sitepodKey := string(sitepod.UID)
	sitepodName := sitepod.Name

	hostname, err := os.Hostname()

	if err != nil {
		glog.Errorf("Unable to get hostname")
		panic(err)
	}

	glog.Infof("Provisioning sitepod %s : %s", sitepodName, sitepod.Spec.DisplayName)

	glog.Info(sitepodKey)
	deploymentObj, err := c.deploymentInformer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		glog.Errorf("Unexpected error getting deployments for %s ", sitepodName)
		return
	}

	// SELECTOR USES SITEPOD RESOUdeploymentEVERSION

	var deployment *ext_api.Deployment

	if len(deploymentObj) == 0 {
		glog.Infof("No existing deployment found for sitepod %s", sitepodName)
		deployment = &ext_api.Deployment{}
		deployment.GenerateName = "sitepod-deployment-"
	} else {
		deployment = deploymentObj[0].(*ext_api.Deployment)
	}

	labels := make(map[string]string)
	labels["sitepod"] = sitepodKey
	deployment.Spec.Replicas = 1
	deployment.Spec.Selector = &unversioned.LabelSelector{MatchLabels: labels}
	deployment.Spec.Template.Spec.NodeName = hostname
	if !(len(deployment.Spec.Template.Spec.Containers) > 1) {
		deployment.Spec.Template.Spec.Containers = []k8s_api.Container{
			k8s_api.Container{
				Name:  "sitepod-alsosleepforever",
				Image: "gcr.io/google_containers/pause:2.0",
			},
		}
	}
	deployment.Spec.Template.GenerateName = "sitepod-pod-"
	deployment.Spec.Template.Labels = labels
	deployment.Labels = labels

	_, err = c.deploymentUpdater(deployment)

	if err != nil {
		glog.Errorf("Requeue - Error adding/updating deployment for sitepod %s: %s", sitepodName, err)
		c.queue.AddAfter(key, RetryDelay)
		return
	}

	// CREATE THE PV
	pvObjs, err := c.pvInformer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		glog.Errorf("Unexpected error getting pvs for %s ", sitepodName)
		return
	}

	var pv *k8s_api.PersistentVolume

	if len(pvObjs) == 0 {
		pv = &k8s_api.PersistentVolume{}
		pv.Annotations = make(map[string]string)
		pv.Annotations["must-provision"] = "true"
	} else {
		pv = pvObjs[0].(*k8s_api.PersistentVolume)
	}

	sitepodDataRoot := "/var/sitepod"
	sitepodDataPath := path.Join(sitepodDataRoot, string(sitepod.UID))

	pv.GenerateName = "sitepod-pv-"
	pv.Spec.AccessModes = []k8s_api.PersistentVolumeAccessMode{k8s_api.ReadWriteOnce}
	pv.Spec.Capacity = make(k8s_api.ResourceList)
	pv.Spec.Capacity[k8s_api.ResourceStorage] = resource.MustParse("1000M")
	pv.Spec.HostPath = &k8s_api.HostPathVolumeSource{}
	pv.Spec.HostPath.Path = sitepodDataPath
	pv.Labels = make(map[string]string)
	pv.Labels["sitepod"] = string(sitepod.UID)
	pv.Labels["hostname"] = hostname

	// TODO eventually must-provision to be handled by NodeJob resource type

	pvObj, err := c.pvUpdater(pv)
	if err != nil {
		glog.Errorf("Error adding/updating new PV for sitepod %s: %s", sitepodName, err)
		c.queue.AddAfter(key, RetryDelay)
		return
	}
	pv = pvObj.(*k8s_api.PersistentVolume)

	if pv.Annotations["must-provision"] == "true" {

		glog.Infof("Creating directory %s", sitepodDataPath)
		err = os.MkdirAll(sitepodDataPath, 0700)
		if err != nil {
			glog.Errorf("Unable to create directory %s: %v", sitepodDataPath, err)
			return
		}
		//create home directory

		err = os.MkdirAll(path.Join(sitepodDataPath, "home"), 0755)
		if err != nil {
			glog.Errorf("Unable to create home directory on %s: %v", sitepodDataPath, err)
			return
		}

		delete(pv.Annotations, "must-provision")
		_, err = c.pvUpdater(pv)
		if err != nil {
			glog.Errorf("Error adding/updating new PV for sitepod %s: %s", sitepodName, err)
			c.queue.AddAfter(key, RetryDelay)
			return
		}
	}

	glog.Infof("Provisioned PV %s", pv.Name)

}

func (c *SitepodController) deleteSitepod(key string) {

	deploymentObjs, err := c.deploymentInformer.GetIndexer().ByIndex("sitepod", key)

	for _, deploymentObj := range deploymentObjs {
		doneDeployment := deploymentObj.(*ext_api.Deployment)
		//err = c.deploymentDeleter(doneDeployment)
		glog.Infof("Deleting deployment %s", doneDeployment.Name)
		if doneDeployment.Spec.Replicas != 0 {
			glog.Infof("Setting replicas to 0 for %s", doneDeployment.Name)
			doneDeployment.Spec.Replicas = 0
			_, err = c.deploymentUpdater(doneDeployment)
			if err != nil {
				glog.Errorf("Unable to set replicates to 0 on deployment: %+v", err)
			}
			time.Sleep(200 * time.Millisecond)
			c.queue.Add(key)
			return
		} else {
			if doneDeployment.Status.Replicas != 0 {
				// TODO use delayed workqueue
				glog.Infof("Replicates not yet 0")
				time.Sleep(200 * time.Millisecond)
				c.queue.Add(key)
			} else {
				glog.Infof("Replicates now yet 0")

				err := c.deploymentDeleter(doneDeployment)
				if err != nil {
					glog.Errorf("Unable to delete deployment")
					return
				}

				selector, err := unversioned.LabelSelectorAsSelector(doneDeployment.Spec.Selector)
				//c.rsSet(labels.Newre
				rsObjs, err := c.rsFilter(selector)
				if err != nil {
					glog.Errorf("Unable to get replica sets %v", err)
					return
				}

				rsList := rsObjs.(*ext_api.ReplicaSetList)
				for _, rsObj := range rsList.Items {
					c.rsDeleter(&rsObj)
				}

			}
		}

	}
}

func (c *SitepodController) HasSynced() bool {
	return c.sitepodInformer.GetController().HasSynced()
}
