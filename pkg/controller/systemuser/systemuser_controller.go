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
	sc := &SystemUserController{*NewSimpleController("SystemUserController", client, []Syncer{client.PVClaims(),
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
		c.Client.SystemUsers().Update(user)
	}

	if !user.Status.HomeProvisioned {

		podTask := c.Client.PodTasks().NewEmpty()
		//TODO this is highly insecure
		podTask.Spec.Command = []string{"mkdir", "-p", "/home/" + user.Spec.Username}
		// TODO chmod

		pod, exists := c.Client.Pods().MaybeSingleBySitepodKey(sitepodKey)
		if !exists {
			return nil
		}

		podTask.Spec.PodName = pod.GetName()
		podTask.Spec.ContainerName = "sitepod-manager"
		podTask.Spec.Namespace = pod.GetNamespace()
		c.Client.PodTasks().Add(podTask)
	}
	return nil

}

func (c *SystemUserController) ProcessDelete(key string) {

	// Determine sitepod belong to
	// Check associated PV
	// Delete user owned directories

	// Leave
}
