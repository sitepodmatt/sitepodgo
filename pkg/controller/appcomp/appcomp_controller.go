package appcomp

// Listen for new app components and build deployments when the sitepod is ready with a root deployment and PV

import (
	"fmt"

	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	k8s_ext "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/controller/framework"
	"sitepod.io/sitepod/pkg/api/v1"
	cc "sitepod.io/sitepod/pkg/client"
	. "sitepod.io/sitepod/pkg/controller/shared"
	"sitepod.io/sitepod/pkg/specgen"
)

type AppCompController struct {
	SimpleController
}

const (
	SpecGenAnnontationKey = "sitepod.io/specgen-onetime"
)

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

	glog.Infof("Processing appcomponent %s", key)
	ac, exists := c.Client.AppComps().MaybeGetByKey(key)

	if !exists {
		glog.Infof("App components %s no longer exists", key)
		return nil
	}

	sitepodKey := ac.Labels["sitepod"]
	_, exists = c.Client.Sitepods().MaybeSingleByUID(sitepodKey)

	if !exists {
		glog.Infof("Sitepod %s no longer exists, skipping app comp %s", sitepodKey, ac.Name)
		return nil
	}

	deployment, exists := c.Client.Deployments().MaybeSingleBySitepodKey(sitepodKey)

	if !exists {
		glog.Errorf("No root deployment exists (yet) for sitepod %s: %s", sitepodKey)
		return DependentResourcesNotReady{fmt.Sprintf("Root deployment for sitepod %s does not yet exist.", sitepodKey)}
	}

	specGenKey := ac.Annotations[SpecGenAnnontationKey]
	if specGenKey == "ssh-server" {
		glog.Infof("Applying spec generation of %s to app comp %s,", specGenKey, ac.Name)
		specgen.SpecGenSSHServer(ac)
		delete(ac.Annotations, SpecGenAnnontationKey)
		// This will requeue the processing with a spec generated in place
		c.Client.AppComps().Update(ac)
		glog.Infof("Updated app comp %s with spec gen", ac.Name)
		return nil
	}

	var destContainer *k8s_api.Container
	destIdx := -1

	for idx, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == ac.Name {
			glog.Infof("Using existing container for app component %s deployment %s", ac.Name, deployment.GetName())
			destContainer = &container
			destIdx = idx
			break
		}
	}

	if destContainer == nil {
		glog.Infof("Creating new container for app component %s deployment %s", ac.Name, deployment.GetName())
		destContainer = &k8s_api.Container{}
		destContainer.Name = ac.Name
	}

	destContainer.Image = fmt.Sprintf("%s:%s", ac.Spec.Image, ac.Spec.ImageVersion)
	destContainer.ImagePullPolicy = k8s_api.PullAlways

	groupedConfigFiles := make(map[string][]v1.AppComponentConfigFile)

	for _, acConfigFile := range ac.Spec.ConfigFiles {
		groupedConfigFiles[acConfigFile.Directory] = append(groupedConfigFiles[acConfigFile.Directory], acConfigFile)
	}

	configMapList := c.Client.ConfigMaps().BySitepodKey(sitepodKey)

	for directory, acConfigFiles := range groupedConfigFiles {

		var matchedConfigMap *k8s_api.ConfigMap
		for _, configMap := range configMapList {
			if configMap.Annotations["sitepod.io/mount-path"] == directory &&
				configMap.Labels["appcomponent"] == ac.Name &&
				configMap.Labels["configtype"] == "appcomponent" {
				matchedConfigMap = configMap
				break
			}
		}

		if matchedConfigMap == nil {
			matchedConfigMap = c.Client.ConfigMaps().NewEmpty()
			matchedConfigMap.Labels = make(map[string]string)
			matchedConfigMap.Data = make(map[string]string)
			matchedConfigMap.Annotations = make(map[string]string)
			matchedConfigMap.Labels["sitepod"] = sitepodKey
			matchedConfigMap.Annotations["sitepod.io/mount-path"] = directory
			matchedConfigMap.Labels["appcomponent"] = ac.Name
			matchedConfigMap.Labels["configtype"] = "appcomponent"
			configMapList = append(configMapList, matchedConfigMap)
		}

		//TODO await PR from rata regarding uid/gid application to configmaps

		keyMap := make(map[string]string)
		for _, acConfigFile := range acConfigFiles {
			matchedConfigMap.Data[acConfigFile.Name] = acConfigFile.Content
			if acConfigFile.Name != acConfigFile.Filename {
				keyMap[acConfigFile.Name] = acConfigFile.Filename
			}
		}

		matchedConfigMap = c.Client.ConfigMaps().UpdateOrAdd(matchedConfigMap)
		c.attachConfigMap(deployment, destContainer, matchedConfigMap, keyMap)
	}

	if ac.Spec.MountEtcs {

		globalConfigMapList := c.Client.ConfigMaps().List()
		for _, configMap := range globalConfigMapList {

			if configMap.Labels["config-type"] != "etc" {
				continue
			}

			c.attachConfigMap(deployment, destContainer, configMap, nil)

		}
	}

	if ac.Spec.MountHome {

		homeVmExists := false
		for _, vm := range destContainer.VolumeMounts {
			if vm.Name == "home-storage" {
				homeVmExists = true
				break
			}
		}

		if !homeVmExists {
			destContainer.VolumeMounts = append(destContainer.VolumeMounts,
				k8s_api.VolumeMount{
					Name:      "home-storage",
					MountPath: "/home",
					SubPath:   "home",
				})
		}
	}

	if destIdx == -1 {
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers,
			*destContainer)
	} else {
		deployment.Spec.Template.Spec.Containers[destIdx] = *destContainer
	}

	if ac.Spec.Expose {

	}

	c.Client.Deployments().Update(deployment)

	return nil
}

func (c *AppCompController) attachConfigMap(deployment *k8s_ext.Deployment, container *k8s_api.Container, cm *k8s_api.ConfigMap, km map[string]string) {

	vmExists := false
	for _, vm := range container.VolumeMounts {
		if vm.Name == cm.Name {
			vmExists = true
			break
		}
	}

	if !vmExists {
		container.VolumeMounts = append(container.VolumeMounts,
			k8s_api.VolumeMount{
				Name:      cm.Name,
				MountPath: cm.Annotations["sitepod.io/mount-path"],
			})
	}

	dvExists := false
	for _, dv := range deployment.Spec.Template.Spec.Volumes {
		if dv.Name == cm.Name {
			dvExists = true
			break
		}
	}

	if !dvExists {

		keyToPaths := []k8s_api.KeyToPath{}
		if km != nil {
			for k, v := range km {
				if len(k) > 0 && len(v) > 0 {
					keyToPaths = append(keyToPaths, k8s_api.KeyToPath{k, v})
				}
			}
		}

		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, k8s_api.Volume{
			Name: cm.Name,
			VolumeSource: k8s_api.VolumeSource{
				ConfigMap: &k8s_api.ConfigMapVolumeSource{
					k8s_api.LocalObjectReference{cm.Name},
					keyToPaths,
				}}})
	}

}
