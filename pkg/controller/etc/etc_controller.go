package etc

// EtcController Listeners for new system users and write out configmaps for underlying etc files

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"github.com/golang/glog"

	"sitepod.io/sitepod/pkg/api/v1"

	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/util/workqueue"
)

var (
	RetryDelay = 200 * time.Millisecond
)

type EtcController struct {
	sitepodInformer    framework.SharedIndexInformer
	systemUserInformer framework.SharedIndexInformer
	configMapGetter    v1.GetterFunc
	configMapUpdater   v1.UpdaterFunc
	queue              workqueue.DelayingInterface
}

func NewEtcController(sitepodInformer framework.SharedIndexInformer,
	systemUserInformer framework.SharedIndexInformer,
	configMapGetter v1.GetterFunc,
	configMapUpdater v1.UpdaterFunc) framework.ControllerInterface {

	c := &EtcController{
		sitepodInformer,
		systemUserInformer,
		configMapGetter,
		configMapUpdater,
		workqueue.NewDelayingQueue(),
	}

	systemUserInformer.AddEventHandler(framework.ResourceEventHandlerFuncs{
		AddFunc:    c.queueAddSystemUser,
		UpdateFunc: c.queueUpdateSystemUser,
		DeleteFunc: c.queueDeleteSystemUser,
	})

	return c
}

var (
	SystemUserEtcs = []string{"users"}
)

func (c *EtcController) Run(stopCh <-chan struct{}) {
	go c.worker()
	<-stopCh
	c.queue.ShutDown()
}

func (c *EtcController) queueAddSystemUser(obj interface{}) {
	user := obj.(*v1.SystemUser)
	if user.Status.AssignedFileUID > 0 {
		for _, etcKey := range SystemUserEtcs {
			c.queue.Add(etcKey)
		}
	} else {
		// We rely on the assignment of file uid will cause a new update event so no requeue
	}
}

func (c *EtcController) queueUpdateSystemUser(old interface{}, cur interface{}) {
	if k8s_api.Semantic.DeepEqual(old, cur) {
		return
	}
	c.queueAddSystemUser(cur)
}

func (c *EtcController) queueDeleteSystemUser(obj interface{}) {
	c.queueAddSystemUser(obj)
}

func (c *EtcController) HasSynced() bool {
	return true
}

func (c *EtcController) worker() {

	if !c.systemUserInformer.HasSynced() {
		glog.Infof("Waiting for system users controller to sync")
		time.Sleep(RetryDelay)
	}

	for {
		func() {
			key, quit := c.queue.Get()
			if quit {
				return
			}
			defer c.queue.Done(key)
			c.syncEtc(key.(string))
		}()
	}
}

func (c *EtcController) syncEtc(key string) {
	startTime := time.Now()
	defer func() {
		glog.V(4).Infof("Finished syncing etc file key %s. (%v)", key, time.Now().Sub(startTime))
	}()

	switch key {
	case "users":
		c.processPasswd()
	default:
		glog.Errorf("Unable to process unexpected key %s", key)
	}

}

func (c *EtcController) processPasswd() {
	passwdContent := []string{}
	shadowContent := []string{}
	suStore := c.systemUserInformer.GetStore()
	systemUserKeys := suStore.ListKeys()
	var err error

	// TODO consider how globally scoped this config map should be
	configObj, err := c.configMapGetter("user-etcs")

	if err != nil {
		if errors.IsNotFound(err) {
			newConfigMap := &k8s_api.ConfigMap{}
			newConfigMap.Name = "user-etcs"
			configObj = newConfigMap
		} else {
			glog.Errorf("Unexpected error get user-etcs config map: %+v", err)
			// NOTE don't requeue here we'll just wait until next full sync
			return
		}
	}

	config := configObj.(*k8s_api.ConfigMap)

	for _, systemUserKey := range systemUserKeys {

		systemUserObj, exists, err := suStore.GetByKey(systemUserKey)

		if err != nil || !exists {
			glog.Errorf("Unable to get system user %s from cache", systemUserKey)
			if err != nil {
				break
			}
			glog.Infof("System user %s not longer exists in cache", systemUserKey)
			continue
		}

		//TODO add a sitepod group - 2000

		user := systemUserObj.(*v1.SystemUser)
		passwdContent = append(passwdContent, fmt.Sprintf("%s:%s:%d:%d:%s:%s:%s\n",
			user.GetUsername(),
			"x", //auth method
			user.Status.AssignedFileUID, //uid
			2000,
			"", //gecos field
			user.GetHomeDirectory(),
			user.GetShell()))

		if user.Spec.Password.IsValid() {
			shadowPassword := fmt.Sprintf("$6$%s$%s",
				user.Spec.Password.Salt,
				user.Spec.Password.CombinedHash)

			shadowContent = append(shadowContent, fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s:%s\n",
				user.GetUsername(), //login name
				shadowPassword,     //encrypted password
				"",                 //date last changed
				"",                 //minimum password age,
				"",                 //maximum password age,
				"",                 //password warning period
				"",                 //password inactivity period
				"",                 //account expiration date
				"",                 //reserved for future use
			))
		}

	}

	passwdOutput, err := processTemplate("etc_passwd", passwdContent)
	if err != nil {
		glog.Errorf("Unable to process passwd template: %+v", err)
		return
	}

	shadowOutput, err := processTemplate("etc_shadow", shadowContent)
	if err != nil {
		glog.Errorf("Unable to process shadow template: %+v", err)
		return
	}

	groupOutput, err := processTemplate("etc_group", []string{})
	if err != nil {
		glog.Errorf("Unable to process group template: %+v", err)
		return
	}

	if config.Labels == nil {
		config.Labels = make(map[string]string)
	}
	config.Labels["config-type"] = "etc"
	if config.Data == nil {
		config.Data = make(map[string]string)
	}
	config.Data["passwd"] = passwdOutput
	config.Data["shadow"] = shadowOutput
	config.Data["group"] = groupOutput

	_, err = c.configMapUpdater(config)

	if err != nil {
		glog.Errorf("Aborting writing etc file due to %+v", err)
		//TODO test if this is an expected optimistic concurrency update conflict?
		c.queue.AddAfter("users", RetryDelay)
	}

}

func processTemplate(path string, data []string) (string, error) {
	template, err := template.ParseFiles("../../templates/" + path)
	if err != nil {
		return "", err
	}
	buffer := bytes.NewBuffer([]byte{})
	err = template.Execute(buffer, data)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
