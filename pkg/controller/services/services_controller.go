package services

// Listen for services and build deployments when the sitepod is ready with a PV
import (
	"bytes"
	"fmt"
	"text/template"

	. "github.com/ahmetalpbalkan/go-linq"
	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/controller/framework"
	"sitepod.io/sitepod/pkg/api/v1"
	cc "sitepod.io/sitepod/pkg/client"
	. "sitepod.io/sitepod/pkg/controller/shared"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
)

type AppCompController struct {
	SimpleController
}

func NewAppCompController(client *cc.Client) framework.ControllerInterface {

	glog.Infof("Creating app component (appcomp) controller")
	c := &AppCompController{*NewSimpleController(client, []Syncer{client.PVClaims(),
		client.PVs(), client.Deployments()}, nil, nil)}
	c.SyncFunc = c.ProcessUpdate
	//sc.DeleteFunc = sc.ProcessDelete
	client.AppComps().AddInformerHandlers(framework.ResourceEventHandlerFuncs{
		AddFunc:    c.QueueAdd,
		UpdateFunc: c.QueueUpdate,
		DeleteFunc: c.QueueDelete,
	})
	return c
}

func (c *AppCompController) QueueAdd(item interface{}) {
	c.EnqueueUpdate(c.Client.AppComps().KeyOf(item))
}

func (c *AppCompController) QueueUpdate(old interface{}, cur interface{}) {
	if !c.Client.AppComps().DeepEqual(old, cur) {
		c.EnqueueUpdate(c.Client.AppComps().KeyOf(cur))
	}
}

func (c *AppCompController) QueueDelete(deleted interface{}) {
	c.EnqueueDelete(c.Client.AppComps().KeyOf(deleted))
}

func (c *AppCompController) ProcessUpdate(key string) error {

	//TODO how we going to handle different app components
	//for now just ssh

	return c.ProcessUpdateSSH(key)
}

func (c *AppCompController) ProcessUpdateSSH(key string) error {

	ac, exists := c.Client.AppComps().MaybeGetByKey(key)

	if !exists {
		glog.Infof("App components %s no longer exists", key)
		return nil
	}

	sitepodKey := ac.Labels["sitepod"]
	sitepod, exists := c.Client.Sitepods().MaybeGetByKey(sitepodKey)

	if !exists {
		glog.Infof("Sitepod %s no longer exists, skipping app comp %s", sitepodKey, ac.Name)
		return nil
	}

	deployment, exists := c.Client.Deployments().MaybeSingleBySitepodKey(sitepodKey)

	if !exists {
		glog.Errorf("No root deployment exists (yet) for sitepod %s: %s", sitepodKey)
		return nil
	}

	sshContainerObj, exists, _ := From(deployment.Spec.Template.Spec.Containers).Where(func(s T) (bool, error) {
		return (s.(k8s_api.Container).Name == "sitepod-ssh"), nil
	}).First()

	// TODO surely we can simplify this
	var sshContainer *k8s_api.Container
	if !exists {
		sshContainer = &k8s_api.Container{}
	} else {
		sshContainer = sshContainerObj.(*k8s_api.Container)
	}

	image := ac.Spec.Image
	version := ac.Spec.ImageVersion
	if image == "" {
		image = "sitepod/sshdftp"
	}
	if version == "" {
		version = "latest"
	}

	sshContainer.Image = image + ":" + version
	sshContainer.ImagePullPolicy = k8s_api.PullAlways
	sshContainer.Name = "sitepod-ssh"

	if ac.Spec.PrivateKeyPEM == "" {
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

		ac.Spec.PrivateKeyPEM = privateKeyPem
		ac.Spec.PublicKeyPEM = fmt.Sprintf("ssh-rsa %s %s", base64.StdEncoding.EncodeToString(pkBytes), "placeholder@sitepod.io")
	}

	//rootStorage := c.Client.PVClaims().SingleBySitepodKey(sitepodKey)

	//FIX this
	homeMounted := false
	for _, vm := range sshContainer.VolumeMounts {
		if vm.Name == "home-storage" {
			homeMounted = true
			break
		}
	}

	if !homeMounted {
		sshContainer.VolumeMounts = append(sshContainer.VolumeMounts, k8s_api.VolumeMount{
			MountPath: "/home",
			Name:      "home-storage",
		})
	}

	//TODO use label selector
	configMapList := c.Client.ConfigMaps().List()

	for _, configMap := range configMapList {

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

		glog.Infof("Config map not found so adding")

		//TODONOW FIX THIS
		sshContainer.VolumeMounts = append(sshContainer.VolumeMounts,
			k8s_api.VolumeMount{
				Name:      configMap.Name,
				MountPath: "/etc/sitepod/etc",
			})

		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, k8s_api.Volume{
			Name: configMap.Name,
			VolumeSource: k8s_api.VolumeSource{
				ConfigMap: &k8s_api.ConfigMapVolumeSource{
					LocalObjectReference: k8s_api.LocalObjectReference{configMap.Name}}}})

	}

	configMaps := c.Client.ConfigMaps().BySitepodKey(sitepodKey)
	sitepodLabel := sitepod.Name

	sshConfigMapName := sitepodLabel + "-" + "sshconfigmap"
	var sshConfigMap *k8s_api.ConfigMap
	for _, configMap := range configMaps {
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

	sshConfigMap.Data["sshdconfig"] = c.generateSshdConfig(ac)
	sshConfigMap.Data["sshhostrsakey"] = ac.Spec.PrivateKeyPEM
	sshConfigMap.Data["sshhostrsakeypub"] = ac.Spec.PublicKeyPEM

	c.Client.ConfigMaps().Update(sshConfigMap)

	sshContainer.VolumeMounts = append(sshContainer.VolumeMounts,
		k8s_api.VolumeMount{
			Name:      "ssh-configmap-volume",
			MountPath: "/etc/sitepod/ssh",
		})

	configMapVolumeInPod := false
	for _, sv := range deployment.Spec.Template.Spec.Volumes {
		if sv.Name == "ssh-configmap-volume" {
			configMapVolumeInPod = true
			break
		}
	}

	if !configMapVolumeInPod {
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes,
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
			})
	}

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
					HostPath: &k8s_api.HostPathVolumeSource{
					//Path: rootStorage.Spec.HostPath.Path + "/home",
					},
				},
			})
	}

	//if isNew {
	////TODONOW test if exist
	//rootDeployment.Spec.Template.Spec.Containers = append(rootDeployment.Spec.Template.Spec.Containers, *sshContainer)
	//}

	//TODONOW: FIX
	c.Client.Deployments().Update(deployment)
	return nil
}

func (c *AppCompController) generateSshdConfig(service *v1.AppComponent) string {
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
