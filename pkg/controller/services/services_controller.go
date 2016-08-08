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
	c := &AppCompController{*NewSimpleController("ServicesController",
		client, []Syncer{client.Sitepods(), client.ConfigMaps(), client.PVClaims(), client.PVs(), client.Deployments()}, nil, nil)}
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
	sitepod, exists := c.Client.Sitepods().MaybeSingleByUID(sitepodKey)

	if !exists {
		glog.Infof("Sitepod %s no longer exists, skipping app comp %s", sitepodKey, ac.Name)
		return nil
	}

	deployment, exists := c.Client.Deployments().MaybeSingleBySitepodKey(sitepodKey)

	if !exists {
		glog.Errorf("No root deployment exists (yet) for sitepod %s: %s", sitepodKey)
		return nil
	}

	//TODO check status of sitepod to ensure provisioned and ready conditions = true!
	pvc := sitepod.Spec.VolumeClaims[0]
	//pv := c.Client.PVs().GetByKey(pvc)

	sshContainerObj, exists, _ := From(deployment.Spec.Template.Spec.Containers).Where(func(s T) (bool, error) {
		return (s.(k8s_api.Container).Name == "sitepod-ssh"), nil
	}).First()

	// TODO surely we can simplify this
	var sshContainer *k8s_api.Container
	isNew := false
	if !exists {
		glog.Infof("Creating new sitepod-ssh container for deployment %s", deployment.GetName())
		sshContainer = &k8s_api.Container{}
		isNew = true
	} else {
		sshContainer = sshContainerObj.(*k8s_api.Container)
		glog.Infof("Found existing sitepod-ssh container for deployment %s", deployment.GetName())
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
			SubPath:   "home",
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

		glog.Infof("VMS: %d", len(sshContainer.VolumeMounts))
		glog.Infof("VMS1: %+v", sshContainer.VolumeMounts[0])
		glog.Infof("Config map %s not found so adding", configMap.Name)

		vmExists, _ := From(sshContainer.VolumeMounts).Where(func(s T) (bool, error) {
			return (s.(k8s_api.VolumeMount).Name == configMap.Name), nil
		}).Any()

		if !vmExists {
			sshContainer.VolumeMounts = append(sshContainer.VolumeMounts,
				k8s_api.VolumeMount{
					Name:      configMap.Name,
					MountPath: "/etc/sitepod/etc",
				})
		}

		dvExists, _ := From(deployment.Spec.Template.Spec.Volumes).Where(func(s T) (bool, error) {
			return (s.(k8s_api.Volume).Name == configMap.Name), nil
		}).Any()

		if !dvExists {
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, k8s_api.Volume{
				Name: configMap.Name,
				VolumeSource: k8s_api.VolumeSource{
					ConfigMap: &k8s_api.ConfigMapVolumeSource{
						LocalObjectReference: k8s_api.LocalObjectReference{configMap.Name}}}})

		}
	}

	glog.Infof("Getting config maps with sitepod key %s", sitepodKey)
	configMaps := c.Client.ConfigMaps().BySitepodKey(sitepodKey)
	glog.Infof("Got %d config maps for sitepod key %s", len(configMaps), sitepodKey)

	sshConfigMapName := sitepod.Name + "-" + "sshconfigmap"
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
		sshConfigMap.Labels["sitepod"] = sitepodKey
	}

	if sshConfigMap.Data == nil {
		sshConfigMap.Data = make(map[string]string)
	}

	sshConfigMap.Data["sshdconfig"] = c.generateSshdConfig(ac)
	sshConfigMap.Data["sshhostrsakey"] = ac.Spec.PrivateKeyPEM
	sshConfigMap.Data["sshhostrsakeypub"] = ac.Spec.PublicKeyPEM

	glog.Info("X")
	c.Client.ConfigMaps().UpdateOrAdd(sshConfigMap)
	glog.Info("X2")

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
					PersistentVolumeClaim: &k8s_api.PersistentVolumeClaimVolumeSource{
						ClaimName: pvc,
					},
				},
			})
	}

	if isNew {
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, *sshContainer)
	}

	//TODONOW: FIX
	c.Client.Deployments().Update(deployment)
	return nil
}

func (c *AppCompController) generateSshdConfig(service *v1.Appcomponent) string {
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
