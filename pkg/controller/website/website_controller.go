package website

import (
	"bytes"
	. "github.com/ahmetalpbalkan/go-linq"
	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/meta"
	//k8s_ext "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/controller/framework"
	"reflect"
	"sitepod.io/sitepod/pkg/api/v1"
	cc "sitepod.io/sitepod/pkg/client"
	. "sitepod.io/sitepod/pkg/controller/shared"
	"text/template"
)

type WebsiteController struct {
	SimpleController
}

func NewWebsiteController(client *cc.Client) framework.ControllerInterface {

	glog.Infof("Creating website controller")
	sc := &WebsiteController{*NewSimpleController("WebsiteController", client,
		[]Syncer{client.ConfigMaps(), client.Sitepods()}, nil, nil)}
	sc.SyncFunc = sc.ProcessUpdate
	client.Websites().AddInformerHandlers(framework.ResourceEventHandlerFuncs{
		AddFunc:    sc.QueueAdd,
		UpdateFunc: sc.QueueUpdate,
		//DeleteFunc: sc.QueueDelete,
	})
	return sc
}

func (c *WebsiteController) Run(stopCh <-chan struct{}) {
	c.SimpleController.Run(stopCh)
}

func (c *WebsiteController) QueueAdd(item interface{}) {
	c.EnqueueUpdate(c.Client.Websites().KeyOf(item))
}

func (c *WebsiteController) QueueUpdate(old interface{}, cur interface{}) {
	c.QueueAdd(cur)
}

func (c *WebsiteController) QueueDelete(deleted interface{}) {
	c.EnqueueDelete(c.Client.Websites().KeyOf(deleted))
	accessor, err := meta.Accessor(deleted)
	if err == nil {
		key := string(accessor.GetUID())
		c.EnqueueDelete(key)
	}
}

func (c *WebsiteController) ProcessUpdate(key string) error {

	glog.Infof("Processing website %s", key)

	website, exists := c.Client.Websites().MaybeGetByKey(key)

	if !exists {
		glog.Infof("Website %s no longer exists", key)
		return nil
	}

	alreadySetup := false
	var err error
	if !website.Status.DirectoryCreated {
		err = c.CreateDirectory(website)
	} else if !website.Status.SkeltonSetup {
		err = c.SkeltonSetup(website)
	} else if !website.Status.ServerSetup {
		err = c.ServerSetup(website)
		//} else if !website.Status.LoadBalancerSetup {
		//err = c.LoadBalancerSetup(website)
	} else {
		alreadySetup = true
	}

	if err != nil {
		glog.Errorf("Error processing website %s: %+v", key, err)
		return err
	}

	if !alreadySetup {
		//c.Client.Websites().Update(website)
	} else {
		glog.Infof("No setup required for website %s", key)
	}

	glog.Infof("Processed website %s", key)
	return nil
}

func (c *WebsiteController) CreateDirectory(website *v1.Website) error {

	sitepodKey := website.Labels["sitepod"]
	podTasks := c.Client.PodTasks().ByIndexByKey("sitepod", sitepodKey)

	cmd := []string{"/bin/mkdir" /* "-p", */, "/home/sitepod/websites/" + website.GetPrimaryDomain()}

	podTaskExists := false
	podTaskExistingPod := ""
	for _, podTask := range podTasks {
		if reflect.DeepEqual(podTask.Spec.Command, cmd) {
			podTaskExists = true
			podTaskExistingPod = podTask.Name
			glog.Infof("Existing podtask for found")
			break
		}
	}

	pod, exists := c.Client.Pods().MaybeSingleBySitepodKey(sitepodKey)
	if !exists {
		return ConditionsNotReady{"Still provisioning pod"}
	}

	if podTaskExists && (podTaskExistingPod == pod.Name) {
		return nil
	}

	readyExists, _ := From(pod.Status.Conditions).Where(func(s T) (bool, error) {
		return (s.(k8s_api.PodCondition).Type == k8s_api.PodReady &&
			s.(k8s_api.PodCondition).Status == k8s_api.ConditionTrue), nil
	}).Any()

	if !readyExists {
		return ConditionsNotReady{"Pod not in ready state"}
	}

	glog.Infof("Creating new pod task")

	podTask := c.Client.PodTasks().NewEmpty()
	podTask.Labels = make(map[string]string)
	podTask.Labels["sitepod"] = sitepodKey
	podTask.Spec.Command = cmd
	podTask.Spec.PodName = pod.GetName()
	podTask.Spec.ContainerName = "sitepod-manager"
	podTask.Spec.Namespace = pod.GetNamespace()
	podTask.Spec.BehalfType = "Website"
	podTask.Spec.BehalfOf = website.Name
	podTask.Spec.BehalfCondition = "DirectoryCreated"
	c.Client.PodTasks().Add(podTask)
	glog.Infof("Created pod task")

	return nil
}

func (c *WebsiteController) SkeltonSetup(website *v1.Website) error {
	website.Status.SkeltonSetup = true
	return nil
}

func (c *WebsiteController) ServerSetup(website *v1.Website) error {

	sitepodKey := website.Labels["sitepod"]

	var webserverConfigMap *k8s_api.ConfigMap

	configMaps := c.Client.ConfigMaps().BySitepodKey(sitepodKey)
	for _, configMap := range configMaps {
		if configMap.Name == "webserver-sites" {
			webserverConfigMap = configMap
			break
		}
	}

	if webserverConfigMap == nil {
		webserverConfigMap = c.Client.ConfigMaps().NewEmpty()
		webserverConfigMap.Labels["sitepod"] = sitepodKey
	}

	confFile := string(website.UID)
	webserverConfigMap.Data[confFile] = processTemplate("nginx-website.conf", website)

	return nil
}

func processTemplate(path string, data interface{}) string {
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
