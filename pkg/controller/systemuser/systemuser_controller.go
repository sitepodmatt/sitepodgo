package systemuser

import (
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/controller/framework"
	cc "sitepod.io/sitepod/pkg/client"
	. "sitepod.io/sitepod/pkg/controller/shared"
)

type SystemUserController struct {
	SimpleController
}

func NewSystemUserController(client *cc.Client) framework.ControllerInterface {

	glog.Info("Creating system user controller")
	sc := &SystemUserController{*NewSimpleController(client, []Syncer{client.PVClaims(),
		client.PVs(), client.Sitepods(), client.SystemUsers()}, nil, nil)}
	sc.SyncFunc = sc.ProcessUpdate
	//sc.DeleteFunc = sc.ProcessDelete
	client.SystemUsers().AddInformerHandlers(framework.ResourceEventHandlerFuncs{
		AddFunc:    sc.QueueAdd,
		UpdateFunc: sc.QueueUpdate,
		DeleteFunc: sc.QueueDelete,
	})
	return sc
}

func (c *SystemUserController) QueueAdd(item interface{}) {
	c.EnqueueUpdate(c.Client.SystemUsers().KeyOf(item))
}

func (c *SystemUserController) QueueUpdate(old interface{}, cur interface{}) {
	if !c.Client.SystemUsers().DeepEqual(old, cur) {
		c.EnqueueUpdate(c.Client.Sitepods().KeyOf(cur))
	}
}

func (c *SystemUserController) QueueDelete(deleted interface{}) {
	c.EnqueueDelete(c.Client.SystemUsers().KeyOf(deleted))
}

func (c *SystemUserController) ProcessUpdate(key string) error {

	user, exists := c.Client.SystemUsers().MaybeGetByKey(key)

	if !exists {
		glog.Infof("User %s no longer exists", key)
		return nil
	}

	glog.Infof("Processing user %s", user.Name)

	sitepodKey := user.Labels["sitepod"]

	_, exists = c.Client.Sitepods().MaybeSingleBySitepodKey(sitepodKey)
	if !exists {
		glog.Infof("Sitepod %s no longer exists, skipping user %s", sitepodKey, user.Name)
		return nil
	}

	if user.Status.AssignedFileUID == 0 {

		cluster, exists := c.Client.Clusters().MaybeGetByKey("sitepod-single-teneant")

		if !exists {
			//TODO pass name
			cluster = c.Client.Clusters().NewEmpty()
		}

		assignedFileUID := cluster.NextFileUID()
		c.Client.Clusters().UpdateOrAdd(cluster)

		user.Status.AssignedFileUID = assignedFileUID
	}

	c.Client.SystemUsers().Update(user)
	return nil

	// TODO find an active pod hosting the PV, send an exec
	// via api-server using websockers

	//if !user.Status.HomeDirCreated {
	////FUTURE: create internalPVjob
	//sitepodKey := user.Labels["sitepod"]

	//if len(sitepodKey) == 0 {
	//glog.Errorf("User %s has no label for sitepod", userName)
	//return
	//}

	//pvObjs, err := c.pvInformer.GetIndexer().ByIndex("sitepod", sitepodKey)

	//if err != nil || len(pvObjs) == 0 {
	//glog.Errorf("Unexpected error unable to get PV for sitepod %s", sitepodKey)
	//return
	//}

	//pv := pvObjs[0].(*k8s_api.PersistentVolume)

	//if pv.Annotations["must-provision"] == "true" {
	//time.Sleep(200 * time.Millisecond)
	//glog.Errorf("Underlying persistent volume not yet ready")
	//c.queue.Add(key)
	//return
	//}

	//rootDataDir := pv.Spec.HostPath.Path
	//homeRoot := "/home/"
	//homeDir := path.Join(rootDataDir, homeRoot, user.GetUsername())

	//_, err = os.Stat(homeDir)
	//if err != nil {
	//err = os.RemoveAll(homeDir)
	//}

	//if err != nil {
	//glog.Errorf("Problem sorting directory: %s", err)
	//return
	//}

	//err = os.Mkdir(homeDir, 0755)
	//if err == nil {
	//err = os.Chown(homeDir, user.Status.AssignedFileUID, 2000)
	//}
	//if err != nil {
	//glog.Errorf("Unable to create home dir for user %s: %v", userName, err)
	//go func() {
	//glog.Infof("Perhaps just recently recreated %s", userName)
	//time.Sleep(1 * time.Second)
	//c.queue.Add(key)
	//}()
	//return
	//}
	//user.Status.HomeDirCreated = true
	//userChanged = true
	//}

}

func (c *SystemUserController) ProcessDelete(key string) {

	// Determine sitepod belong to
	// Check associated PV
	// Delete user owned directories

	// Leave
}
