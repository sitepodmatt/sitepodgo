package systemuser

import (
	. "github.com/ahmetalpbalkan/go-linq"
	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/controller/framework"
	"reflect"
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

	_, exists = c.Client.Sitepods().MaybeSingleByUID(sitepodKey)
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

		podTasks := c.Client.PodTasks().ByIndexByKey("sitepod", sitepodKey)

		cmd := []string{"/bin/mkdir", "-p", "/home/" + user.GetUsername()}

		podTaskExists := false
		for _, podTask := range podTasks {
			if reflect.DeepEqual(podTask.Spec.Command, cmd) {
				podTaskExists = true
				glog.Infof("Existing podtask for home dir creation for %s on %s found", key, sitepodKey)
				break
			}
		}

		if !podTaskExists {
			glog.Infof("Creating job to build home directory for %s", key)
			// TODO CHECK if podtask is exists

			podTask := c.Client.PodTasks().NewEmpty()
			//TODO this is highly insecure
			podTask.Spec.Command = cmd
			// TODO chmod

			pod, exists := c.Client.Pods().MaybeSingleBySitepodKey(sitepodKey)
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

			podTask.Labels = make(map[string]string)
			podTask.Labels["sitepod"] = sitepodKey
			podTask.Spec.PodName = pod.GetName()
			podTask.Spec.ContainerName = "sitepod-manager"
			podTask.Spec.Namespace = pod.GetNamespace()
			podTask.Spec.BehalfType = "SystemUser"
			podTask.Spec.BehalfOf = user.Name
			podTask.Spec.BehalfCondition = "HomeProvisioned"
			c.Client.PodTasks().Add(podTask)
			glog.Infof("Created job to build home directory for %s", key)
		}
	}
	return nil

}

func (c *SystemUserController) ProcessDelete(key string) {

	// Determine sitepod belong to
	// Check associated PV
	// Delete user owned directories

	// Leave
}
