package services

// Listen for services and build deployments when the sitepod is ready with a PV

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"golang.org/x/crypto/ssh"

	"github.com/golang/glog"

	"sitepod.io/sitepod/pkg/api/v1"

	k8s_api "k8s.io/kubernetes/pkg/api"
	ext_api "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/util/workqueue"
)

type ServicesController struct {
	servicesInformer    framework.SharedIndexInformer
	sitepodInformer     framework.SharedIndexInformer
	deploymentsInformer framework.SharedIndexInformer
	pvInformer          framework.SharedIndexInformer
	configMapInformer   framework.SharedIndexInformer
	deploymentUpdater   v1.UpdaterFunc
	configMapUpdater    v1.UpdaterFunc
	queue               workqueue.Interface
}

func NewServicesController(servicesInformer framework.SharedIndexInformer,
	sitepodInformer framework.SharedIndexInformer,
	deploymentsInformer framework.SharedIndexInformer,
	pvInformer framework.SharedIndexInformer,
	configMapInformer framework.SharedIndexInformer,
	deploymentUpdater v1.UpdaterFunc,
	configMapUpdater v1.UpdaterFunc) framework.ControllerInterface {

	c := &ServicesController{servicesInformer,
		sitepodInformer,
		deploymentsInformer,
		pvInformer,
		configMapInformer,
		deploymentUpdater,
		configMapUpdater,
		workqueue.New(),
	}

	servicesInformer.AddEventHandler(framework.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
		DeleteFunc: c.deleteService})

	return c
}

func (c *ServicesController) addService(obj interface{}) {
	service := obj.(*v1.Serviceinstance)
	key, _ := v1.DefaultKeyFunc(service)
	c.queue.Add(key)
}

func (c *ServicesController) updateService(old interface{}, cur interface{}) {
	if k8s_api.Semantic.DeepEqual(old, cur) {
		return
	}
	service := cur.(*v1.Serviceinstance)
	key, _ := v1.DefaultKeyFunc(service)
	c.queue.Add(key)
}

func (c *ServicesController) deleteService(obj interface{}) {
	//TODO handle this seperately
}

func (c *ServicesController) Run(stopCh <-chan struct{}) {
	go c.worker()
	<-stopCh
	c.queue.ShutDown()
}

func (c *ServicesController) worker() {
	for {
		func() {
			key, quit := c.queue.Get()
			if quit {
				return
			}
			defer c.queue.Done(key)
			c.syncService(key.(string))
		}()
	}
}

func (c *ServicesController) IsReady() bool {
	return (c.servicesInformer.HasSynced() && c.sitepodInformer.HasSynced() &&
		c.deploymentsInformer.HasSynced() && c.configMapInformer.HasSynced())
}

func (c *ServicesController) syncService(key string) {

	for !c.IsReady() {
		time.Sleep(50 * time.Millisecond)
		c.queue.Add(key)
		return
	}

	obj, exists, err := c.servicesInformer.GetStore().GetByKey(key)

	if err != nil {
		glog.Errorf("Error getting service: %+v", err)
		return
	}

	if !exists {
		glog.Errorf("Service %s no longer exists", key)
		return
	}

	glog.Infof("Processing service instance %s", key)
	service := obj.(*v1.Serviceinstance)

	if service.Spec.Type == "ssh" {
		c.syncSSHService(service)
	} else {
		glog.Errorf("Unsupported service %s for service instance %s", service.Spec.Type, service.Name)
	}

}

func (c *ServicesController) syncSSHService(service *v1.Serviceinstance) {

	sitepodLabel := service.Labels["sitepod"]

	if sitepodLabel == "" {
		glog.Errorf("Unexpected no sitepod label for service %s", service.Name)
		return
	}

	sitepodObjs, err := c.sitepodInformer.GetIndexer().ByIndex("uid", sitepodLabel)

	if err != nil {
		glog.Errorf("Unexpected err getting sitepod %s for service %s: %s", sitepodLabel, service.Name, err)
		return
	}

	if len(sitepodObjs) == 0 {
		glog.Errorf("%v", c.sitepodInformer.GetStore().ListKeys())
		glog.Errorf("%v", c.sitepodInformer.GetIndexer().ListKeys())
		glog.Errorf("Non-existant sitepod %s for service %s", sitepodLabel, service.Name)
		return
	}

	sitepod := sitepodObjs[0].(*v1.Sitepod)

	sitepodKey := string(sitepod.UID)
	sitepodName := sitepod.Name

	rootDeploymentObj, err := c.deploymentsInformer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		glog.Errorf("Error getting root deployment sitepod %s: %s", sitepodName, err)
		return
	}

	// If we get a root deployment we presume all configs and data storage pvs are setup
	if len(rootDeploymentObj) == 0 {
		glog.Errorf("No root deployment exists (yet) for sitepod %s: %s", sitepodName, err)
		return
	}

	rootDeployment := rootDeploymentObj[0].(*ext_api.Deployment)

	var sshContainer *k8s_api.Container
	isNew := false
	glog.Infof("Existing containers %d", len(rootDeployment.Spec.Template.Spec.Containers))
	for _, container := range rootDeployment.Spec.Template.Spec.Containers {
		if container.Name == "sitepod-ssh" {
			glog.Infof("Updating existing ssh container for sitepod %s", sitepod.Name)
			sshContainer = &container
			break
		}
	}

	if sshContainer == nil {
		glog.Infof("Creating new ssh container for sitepod %s", sitepod.Name)
		isNew = true
		sshContainer = &k8s_api.Container{}
	}
	//rootPerm := int64(104)
	//rootDeployment.Spec.Template.Spec.SecurityContext.FSGroup = &rootPerm
	image := service.Spec.Image
	version := service.Spec.ImageVersion
	if image == "" {
		image = "sitepod/sshdftp"
	}
	if version == "" {
		version = "latest"
	}
	sshContainer.Image = image + ":" + version
	sshContainer.ImagePullPolicy = k8s_api.PullAlways

	//sshContainer.Command = []string{"/bin/bash"}
	//sshContainer.Args = []string{"-c", "sleep 100d"}

	sshContainer.Name = "sitepod-ssh"

	if service.Spec.PrivateKeyPEM == "" {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2014)
		if err != nil {
			panic(err)
		}

		privateKeyDer := x509.MarshalPKCS1PrivateKey(privateKey)
		privateKeyBlock := pem.Block{
			Type:    "RSA PRIVATE KEY",
			Headers: nil,
			Bytes:   privateKeyDer,
		}

		privateKeyPem := string(pem.EncodeToMemory(&privateKeyBlock))

		publicKey := privateKey.PublicKey

		pub, err := ssh.NewPublicKey(&publicKey)

		if err != nil {
			panic(err)
		}

		pkBytes := pub.Marshal()

		service.Spec.PrivateKeyPEM = privateKeyPem
		service.Spec.PublicKeyPEM = fmt.Sprintf("ssh-rsa %s %s", base64.StdEncoding.EncodeToString(pkBytes), "placeholder@sitepod.io")
	}

	rootStorageObj, err := c.pvInformer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		glog.Errorf("Unexpected err getting root pv for sitepod", sitepodName, err)
		return
	}

	if len(rootStorageObj) == 0 {
		glog.Errorf("Non-existant root pv for sitepod %s", sitepodName)
		return
	}

	rootStorage := rootStorageObj[0].(*k8s_api.PersistentVolume)

	sshContainer.VolumeMounts = []k8s_api.VolumeMount{
		k8s_api.VolumeMount{
			MountPath: "/home",
			Name:      "home-storage",
			//SubPath:   "home", //TODO this isn't working?  Kubernetes issue 26986
		},
	}

	//TODO use label selector
	configMapList := c.configMapInformer.GetStore().List()

	if err != nil {
		glog.Errorf("Unable to get config maps for sitepod %s: %s", sitepodName, err)
		return
	}

	for _, v := range configMapList {
		configMap := v.(*k8s_api.ConfigMap)

		if configMap.Labels["config-type"] != "etc" {
			continue
		}

		skipMount := false
		for _, vm := range sshContainer.VolumeMounts {
			if vm.Name == configMap.Name {
				skipMount = true
				break
			}
		}

		if skipMount {
			continue
		}

		//TODONOW FIX THIS
		sshContainer.VolumeMounts = append(sshContainer.VolumeMounts,
			k8s_api.VolumeMount{
				Name:      configMap.Name,
				MountPath: "/etc/sitepod/etc",
			})

		rootDeployment.Spec.Template.Spec.Volumes = append(rootDeployment.Spec.Template.Spec.Volumes, k8s_api.Volume{
			Name: configMap.Name,
			VolumeSource: k8s_api.VolumeSource{
				ConfigMap: &k8s_api.ConfigMapVolumeSource{
					LocalObjectReference: k8s_api.LocalObjectReference{configMap.Name}}}})

	}

	// if spec empty generate privatekey, public key

	// save if changed

	// generate Config Maps

	configMaps, err := c.configMapInformer.GetIndexer().ByIndex("sitepod", sitepodLabel)

	if err != nil {
		glog.Errorf("Unable to get config maps for %s", sitepodLabel)
		return
	}

	sshConfigMapName := sitepodLabel + "-" + "sshconfigmap"
	var sshConfigMap *k8s_api.ConfigMap
	for _, configMapObj := range configMaps {
		configMap := configMapObj.(*k8s_api.ConfigMap)
		if configMap.GetName() == sshConfigMapName {
			sshConfigMap = configMap
			break
		}
	}

	if sshConfigMap == nil {
		sshConfigMap = &k8s_api.ConfigMap{}
		sshConfigMap.Name = sshConfigMapName
		sshConfigMap.Labels = make(map[string]string)
		sshConfigMap.Labels["sitepod"] = sitepodLabel
	}

	if sshConfigMap.Data == nil {
		sshConfigMap.Data = make(map[string]string)
	}

	sshConfigMap.Data["sshdconfig"] = c.generateSshdConfig(service)
	sshConfigMap.Data["sshhostrsakey"] = service.Spec.PrivateKeyPEM
	sshConfigMap.Data["sshhostrsakeypub"] = service.Spec.PublicKeyPEM

	_, err = c.configMapUpdater(sshConfigMap)

	if err != nil {
		glog.Errorf("Unable to update config map %s: %v", sshConfigMapName, err)
		return
	}

	sshContainer.VolumeMounts = append(sshContainer.VolumeMounts,
		k8s_api.VolumeMount{
			Name:      "ssh-configmap-volume",
			MountPath: "/etc/sitepod/ssh",
		})

	rootDeployment.Spec.Template.Spec.Volumes = append(rootDeployment.Spec.Template.Spec.Volumes,
		k8s_api.Volume{
			Name: "ssh-configmap-volume",
			VolumeSource: k8s_api.VolumeSource{
				ConfigMap: &k8s_api.ConfigMapVolumeSource{
					k8s_api.LocalObjectReference{sshConfigMapName},
					[]k8s_api.KeyToPath{
						k8s_api.KeyToPath{"sshdconfig", "sshd_config"},
						k8s_api.KeyToPath{"sshhostrsakey", "ssh_host_rsa_key"},
						k8s_api.KeyToPath{"sshhostrsakeypub", "ssh_host_rsa_key.pub"},
					}},
			},
		},
		//TODO: This is a hack, use claims and properly utilize PersistenntVolumes
		k8s_api.Volume{
			Name: "home-storage",
			VolumeSource: k8s_api.VolumeSource{
				HostPath: &k8s_api.HostPathVolumeSource{
					Path: rootStorage.Spec.HostPath.Path + "/home",
				},
			},
		},
	)

	if isNew {
		//TODONOW test if exist
		rootDeployment.Spec.Template.Spec.Containers = append(rootDeployment.Spec.Template.Spec.Containers, *sshContainer)
	}

	//TODONOW: FIX
	glog.Infof("Updating deployment %s", rootDeployment.Name)
	_, err = c.deploymentUpdater(rootDeployment)
	if err != nil {
		glog.Errorf("Unable to update rc %s: %s", rootDeployment.Name, err)
		return
	}

}

func (c *ServicesController) HasSynced() bool {
	return c.sitepodInformer.GetController().HasSynced()
}

func (c *ServicesController) generateSshdConfig(service *v1.Serviceinstance) string {
	template, err := template.ParseFiles("../../templates/sshd_config")
	if err != nil {
		panic(err)
	}
	buffer := bytes.NewBuffer([]byte{})
	err = template.Execute(buffer, struct{}{})
	if err != nil {
		panic(err)
	}
	return buffer.String()
}
