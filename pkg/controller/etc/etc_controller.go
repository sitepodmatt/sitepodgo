package etc

// EtcController Listeners for new system users and write out configmaps for underlying etc files

import (
	"bytes"
	"fmt"
	"text/template"

	//. "github.com/ahmetalpbalkan/go-linq"
	"github.com/golang/glog"
	//k8s_api "k8s.io/kubernetes/pkg/api"
	//kerrors "k8s.io/kubernetes/pkg/api/errors"
	//"k8s.io/kubernetes/pkg/api/unversioned"
	//ext_api "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/controller/framework"
	//"sitepod.io/sitepod/pkg/api/v1"
	cc "sitepod.io/sitepod/pkg/client"
	. "sitepod.io/sitepod/pkg/controller/shared"
)

type EtcController struct {
	SimpleController
}

func NewEtcController(client *cc.Client) framework.ControllerInterface {

	glog.Infof("Creating etc controller")
	sc := &EtcController{*NewSimpleController("EtcController", client, []Syncer{client.SystemUsers(),
		client.ConfigMaps(), client.Sitepods()}, nil, nil)}
	sc.SyncFunc = sc.ProcessUpdate
	client.SystemUsers().AddInformerHandlers(framework.ResourceEventHandlerFuncs{
		AddFunc:    sc.QueueAdd,
		UpdateFunc: sc.QueueUpdate,
		DeleteFunc: sc.QueueDelete,
	})
	return sc
}

var (
	SystemUserEtcs = []string{"users"}
)

func (c *EtcController) Run(stopCh <-chan struct{}) {
	// We want to fire a run even if there are no system users
	c.QueueAdd(struct{}{})
	c.SimpleController.Run(stopCh)
}

func (c *EtcController) QueueAdd(item interface{}) {
	for _, etcKey := range SystemUserEtcs {
		c.EnqueueUpdate(etcKey)
	}
}

func (c *EtcController) QueueUpdate(old interface{}, cur interface{}) {
	if !c.Client.SystemUsers().DeepEqual(old, cur) {
		c.QueueAdd(cur)
	}
}

func (c *EtcController) QueueDelete(deleted interface{}) {
	// we rewrite etc configmaps when a user is removed so this similar as add/update
	c.QueueAdd(deleted)
}

func (c *EtcController) ProcessUpdate(key string) error {

	glog.Info("Rebuilding passwd and shadow content")
	passwdContent := []string{}
	shadowContent := []string{}

	systemUsers := c.Client.SystemUsers().List()

	config, exists := c.Client.ConfigMaps().MaybeGetByKey("user-etcs")
	if !exists {
		//TODO NewEmpty to accept minimum name or generateName?
		config = c.Client.ConfigMaps().NewEmpty()
		config.Name = "user-etcs"
		glog.Info("Creating new config map user-etcs")
	} else {
		glog.Infof("Using existing  config map %s : %s", string(config.UID), config.GetName())
	}

	glog.Infof("Building config with %d system users", len(systemUsers))
	for _, user := range systemUsers {

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

	passwdOutput := processTemplate("etc_passwd", passwdContent)
	shadowOutput := processTemplate("etc_shadow", shadowContent)
	groupOutput := processTemplate("etc_group", []string{})

	// Move to defaulters
	if config.Labels == nil {
		config.Labels = make(map[string]string)
	}
	if config.Data == nil {
		config.Data = make(map[string]string)
	}

	if config.Annotations == nil {
		config.Annotations = make(map[string]string)
	}
	config.Annotations["sitepod.io/mount-path"] = "/etc/sitepod/etc"
	config.Labels["config-type"] = "etc"
	config.Data["passwd"] = passwdOutput
	config.Data["shadow"] = shadowOutput
	config.Data["group"] = groupOutput

	glog.Infof("Updating config map %s", config.GetName())
	c.Client.ConfigMaps().UpdateOrAdd(config)
	glog.Infof("Updated config map %s", config.GetName())

	return nil
}

func processTemplate(path string, data []string) string {
	template, err := template.ParseFiles("../../templates/" + path)
	if err != nil {
		panic(err)
	}

	buffer := bytes.NewBuffer([]byte{})
	err = template.Execute(buffer, data)
	if err != nil {
		panic(err)
	}
	return buffer.String()
}
