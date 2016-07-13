package systemuser

import (
	"os"
	"path"
	"time"

	"github.com/golang/glog"

	"sitepod.io/sitepod/pkg/api/v1"

	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/util/workqueue"
)

type SystemUserController struct {
	sitepodInformer    framework.SharedIndexInformer
	systemUserInformer framework.SharedIndexInformer
	pvInformer         framework.SharedIndexInformer
	clusterGetter      v1.GetterFunc
	clusterUpdater     v1.UpdaterFunc
	systemUserUpdater  v1.UpdaterFunc
	queue              workqueue.Interface
	deleteMap          map[string]systemUserDeleteMarker
}

type systemUserDeleteMarker struct {
	Username   string
	SitepodKey string
}

func NewSystemUserController(sitepodInformer framework.SharedIndexInformer,
	systemUserInformer framework.SharedIndexInformer,
	pvInformer framework.SharedIndexInformer,
	clusterGetter v1.GetterFunc,
	clusterUpdater v1.UpdaterFunc,
	systemUserUpdater v1.UpdaterFunc) framework.ControllerInterface {

	c := &SystemUserController{
		sitepodInformer,
		systemUserInformer,
		pvInformer,
		clusterGetter,
		clusterUpdater,
		systemUserUpdater,
		workqueue.New(),
		make(map[string]systemUserDeleteMarker),
	}

	systemUserInformer.AddEventHandler(framework.ResourceEventHandlerFuncs{
		AddFunc:    c.addSystemUser,
		UpdateFunc: c.updateSystemUser,
		DeleteFunc: c.deleteSystemUser, //we dont care about deletions, or reclaiming file uids
	})

	return c
}

func (c *SystemUserController) addSystemUser(obj interface{}) {
	user := obj.(*v1.SystemUser)
	key, _ := v1.DefaultKeyFunc(user)
	c.queue.Add(key)
}

func (c *SystemUserController) updateSystemUser(old interface{}, cur interface{}) {
	if k8s_api.Semantic.DeepEqual(old, cur) {
		return
	}
	user := cur.(*v1.SystemUser)
	key, _ := v1.DefaultKeyFunc(user)
	c.queue.Add(key)
}

func (c *SystemUserController) deleteSystemUser(obj interface{}) {
	if user, ok := obj.(*v1.SystemUser); ok {
		key, _ := v1.DefaultKeyFunc(user)
		//TODO: Is there a better way to handle this as user will likely be gone from store
		// by time we get to it
		c.deleteMap[key] = systemUserDeleteMarker{
			Username:   user.GetUsername(),
			SitepodKey: user.Labels["sitepod"]}
		c.queue.Add(key)
	} else {
		//TODO: Can we ever get to this branch?
	}
}

func (c *SystemUserController) Run(stopCh <-chan struct{}) {
	go c.worker()
	<-stopCh
	c.queue.ShutDown()
}

func (c *SystemUserController) HasSynced() bool {
	return c.pvInformer.HasSynced()
}

func (c *SystemUserController) worker() {
	for {
		func() {
			key, quit := c.queue.Get()
			if quit {
				return
			}
			defer c.queue.Done(key)
			if _, exists, _ := c.systemUserInformer.GetStore().GetByKey(key.(string)); exists {
				c.syncSystemUser(key.(string))
			} else {
				c.syncDeleteSystemUser(key.(string))
			}
		}()
	}
}

func (c *SystemUserController) IsReady() bool {
	return (c.sitepodInformer.HasSynced() && c.pvInformer.HasSynced())
}

func (c *SystemUserController) syncSystemUser(key string) {

	for !c.IsReady() {
		glog.Info("Waiting for deployment informer and pv informer to sync")
		time.Sleep(50 * time.Millisecond)
	}

	systemUserObj, exists, err := c.systemUserInformer.GetStore().GetByKey(key)
	if err != nil {
		glog.Errorf("Can not get user %s : %s", key, err)
		return
	}

	if !exists {
		glog.Infof("User %s no longer exists", key)
		return
	}

	user := systemUserObj.(*v1.SystemUser)
	userName := user.Name
	userChanged := false

	glog.Infof("Processing user %s", userName)

	sitepodLabel := user.Labels["sitepod"]

	if sitepodLabel == "" {
		glog.Errorf("Unexpected no sitepod label for user %s", userName)
		return
	}

	sitepodObjs, err := c.sitepodInformer.GetIndexer().ByIndex("uid", sitepodLabel)

	if err != nil {
		glog.Errorf("Unexpected err getting sitepod %s for user %s: %s", sitepodLabel, userName, err)
		return
	}

	if len(sitepodObjs) == 0 {
		glog.Errorf("Non-existant sitepod  %s for service %s", sitepodLabel, userName)
		return
	}

	if user.Status.AssignedFileUID == 0 {
		clusterObj, err := c.clusterGetter("singlenode")

		if err != nil {
			glog.Errorf("Unable to get cluster singlenode for %s", userName)
			c.queue.Add(key)
			return
		}

		cluster := clusterObj.(*v1.Cluster)
		assignedFileUID := cluster.NextFileUID()
		// Lock in the number now incase of failure or retry
		_, err = c.clusterUpdater(cluster)
		if err != nil {
			//retry if optimistic currency rejection
			glog.Errorf("Unable to update cluster singlenode for %s", userName)
			c.queue.Add(key)
			return
		}
		user.Status.AssignedFileUID = assignedFileUID
		glog.Infof("Assigned %s new File UID %d", userName, assignedFileUID)
		userChanged = true
	}

	if !user.Status.HomeDirCreated {
		//FUTURE: create internalPVjob
		sitepodKey := user.Labels["sitepod"]

		if len(sitepodKey) == 0 {
			glog.Errorf("User %s has no label for sitepod", userName)
			return
		}

		pvObjs, err := c.pvInformer.GetIndexer().ByIndex("sitepod", sitepodKey)

		if err != nil || len(pvObjs) == 0 {
			glog.Errorf("Unexpected error unable to get PV for sitepod %s", sitepodKey)
			return
		}

		pv := pvObjs[0].(*k8s_api.PersistentVolume)

		if pv.Annotations["must-provision"] == "true" {
			glog.Errorf("Underlying persistent volume not yet ready")
			c.queue.Add(key)
			return
		}

		rootDataDir := pv.Spec.HostPath.Path
		homeRoot := "/home/"
		homeDir := path.Join(rootDataDir, homeRoot, user.GetUsername())

		_, err = os.Stat(homeDir)
		if err != nil {
			err = os.RemoveAll(homeDir)
		}

		if err != nil {
			glog.Errorf("Problem sorting directory: %s", err)
			return
		}

		err = os.Mkdir(homeDir, 0755)
		if err == nil {
			err = os.Chown(homeDir, user.Status.AssignedFileUID, 2000)
		}
		if err != nil {
			glog.Errorf("Unable to create home dir for user %s: %v", userName, err)
			go func() {
				glog.Infof("Perhaps just recently recreated %s", userName)
				time.Sleep(1 * time.Second)
				c.queue.Add(key)
			}()
			return
		}
		user.Status.HomeDirCreated = true
		userChanged = true
	}

	if userChanged {
		glog.Infof("Updating user %s", userName)
		_, err := c.systemUserUpdater(user)
		if err != nil {
			glog.Errorf("Unable to update user")
			c.queue.Add(user)
		}
	}

}

func (c *SystemUserController) syncDeleteSystemUser(key string) {

	for !c.IsReady() {
		glog.Info("Waiting for deployment informer and pv informer to sync")
		time.Sleep(50 * time.Millisecond)
	}

	if deleteMarker, found := c.deleteMap[key]; found {

		pvObjs, err := c.pvInformer.GetIndexer().ByIndex("sitepod", deleteMarker.SitepodKey)

		if err != nil || len(pvObjs) == 0 {
			glog.Errorf("Unexpected error unable to get PV for sitepod %s", deleteMarker.SitepodKey)
			return
		}

		pv := pvObjs[0].(*k8s_api.PersistentVolume)
		rootDataDir := pv.Spec.HostPath.Path
		homeRoot := "/home/"
		homeDir := path.Join(rootDataDir, homeRoot, deleteMarker.Username)

		err = os.RemoveAll(homeDir)

		if err != nil {
			glog.Errorf("Error removing %s", deleteMarker.Username)
		}

	}
}
