package client

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	k8s_v1 "k8s.io/kubernetes/pkg/api/v1"
	ext_v1 "k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/runtime/serializer"
	"sync"
)

type ClientConfig struct {
	ApiServer string
	Namespace string
}

type Client struct {
	config                  *ClientConfig
	scheme                  *runtime.Scheme
	serializer              runtime.NegotiatedSerializer
	sitepodRestClient       *restclient.RESTClient
	k8sCoreRestClient       *restclient.RESTClient
	k8sExtRestClient        *restclient.RESTClient
	sitepodRestClientConfig *restclient.Config
	k8sCoreRestClientConfig *restclient.Config
	k8sExtRestClientConfig  *restclient.Config
	cachedClients           map[string]interface{}
	cachedClientMutex       sync.Mutex
}

func NewClient(scheme *runtime.Scheme, config *ClientConfig) *Client {
	c := &Client{scheme: scheme, config: config}
	c.cachedClients = make(map[string]interface{})
	c.serializer = serializer.NewCodecFactory(scheme)

	sitepodGroupVersion := &unversioned.GroupVersion{"stable.sitepod.io", "v1"}
	c.sitepodRestClient, c.sitepodRestClientConfig = c.buildRestClient("apis", sitepodGroupVersion)
	c.k8sCoreRestClient, c.k8sCoreRestClientConfig = c.buildRestClient("api", &k8s_v1.SchemeGroupVersion)
	c.k8sExtRestClient, c.k8sExtRestClientConfig = c.buildRestClient("apis", &ext_v1.SchemeGroupVersion)

	return c
}

func (c *Client) usingCache(key string, fn func() interface{}) interface{} {

	c.cachedClientMutex.Lock()
	if c.cachedClients[key] == nil {
		c.cachedClients[key] = fn()
	}
	c.cachedClientMutex.Unlock()
	return c.cachedClients[key]
}

func (c *Client) Sitepods() *SitepodClient {
	return c.usingCache("sitepods", func() interface{} {
		return NewSitepodClient(c.sitepodRestClient, c.sitepodRestClientConfig, c.config.Namespace)
	}).(*SitepodClient)
}

func (c *Client) PVClaims() *PVClaimClient {
	return c.usingCache("pvclaims", func() interface{} {
		return NewPVClaimClient(c.k8sCoreRestClient, c.k8sCoreRestClientConfig, c.config.Namespace)
	}).(*PVClaimClient)
}

func (c *Client) PVs() *PVClient {
	return c.usingCache("pvs", func() interface{} {
		return NewPVClient(c.k8sCoreRestClient, c.k8sCoreRestClientConfig, c.config.Namespace)
	}).(*PVClient)
}

func (c *Client) Pods() *PodClient {
	return c.usingCache("pods", func() interface{} {
		return NewPodClient(c.k8sCoreRestClient, c.k8sCoreRestClientConfig, c.config.Namespace)
	}).(*PodClient)
}

func (c *Client) Deployments() *DeploymentClient {
	return c.usingCache("deployments", func() interface{} {
		return NewDeploymentClient(c.k8sExtRestClient, c.k8sExtRestClientConfig, c.config.Namespace)
	}).(*DeploymentClient)
}

func (c *Client) ReplicaSets() *ReplicaSetClient {
	return c.usingCache("replicasets", func() interface{} {
		return NewReplicaSetClient(c.k8sExtRestClient, c.k8sExtRestClientConfig, c.config.Namespace)
	}).(*ReplicaSetClient)
}

func (c *Client) SystemUsers() *SystemUserClient {
	return c.usingCache("systemusers", func() interface{} {
		return NewSystemUserClient(c.sitepodRestClient, c.sitepodRestClientConfig, c.config.Namespace)
	}).(*SystemUserClient)
}

func (c *Client) ConfigMaps() *ConfigMapClient {
	return c.usingCache("configmaps", func() interface{} {
		return NewConfigMapClient(c.k8sCoreRestClient, c.k8sCoreRestClientConfig, c.config.Namespace)
	}).(*ConfigMapClient)
}

func (c *Client) Clusters() *ClusterClient {
	return c.usingCache("clusters", func() interface{} {
		return NewClusterClient(c.sitepodRestClient, c.sitepodRestClientConfig, c.config.Namespace)
	}).(*ClusterClient)
}

func (c *Client) AppComps() *AppCompClient {
	return c.usingCache("appcomps", func() interface{} {
		return NewAppCompClient(c.sitepodRestClient, c.sitepodRestClientConfig, c.config.Namespace)
	}).(*AppCompClient)
}

func (c *Client) PodTasks() *PodTaskClient {
	return c.usingCache("podtasks", func() interface{} {
		return NewPodTaskClient(c.sitepodRestClient, c.sitepodRestClientConfig, c.config.Namespace)
	}).(*PodTaskClient)
}

func (c *Client) Services() *ServiceClient {
	return c.usingCache("servicesc", func() interface{} {
		return NewServiceClient(c.k8sCoreRestClient, c.k8sCoreRestClientConfig, c.config.Namespace)
	}).(*ServiceClient)
}

func (c *Client) Websites() *WebsiteClient {
	return c.usingCache("websites", func() interface{} {
		return NewWebsiteClient(c.sitepodRestClient, c.sitepodRestClientConfig, c.config.Namespace)
	}).(*WebsiteClient)
}

func (c *Client) buildRestClient(apiPath string, gv *unversioned.GroupVersion) (*restclient.RESTClient, *restclient.Config) {

	rcConfig := &restclient.Config{
		Host:    c.config.ApiServer,
		APIPath: apiPath,
		ContentConfig: restclient.ContentConfig{
			GroupVersion:         gv,
			NegotiatedSerializer: c.serializer,
		},
	}

	rc, err := restclient.RESTClientFor(rcConfig)

	if err != nil {
		panic(err)
	}

	return rc, rcConfig
}

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" SitepodClient(v1.Sitepod,v1.SitepodList,"Sitepod","Sitepods",true,"sitepod-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" PVClaimClient(k8s_api.PersistentVolumeClaim,k8s_api.PersistentVolumeClaimList,"PersistentVolumeClaim","PersistentVolumeClaims",true,"sitepod-pvc-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" PVClient(k8s_api.PersistentVolume,k8s_api.PersistentVolumeList,"PersistentVolume","PersistentVolumes",false,"sitepod-pv-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" PodClient(k8s_api.Pod,k8s_api.PodList,"Pod","Pods",true,"sitepod-pod-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" DeploymentClient(ext_api.Deployment,ext_api.DeploymentList,"Deployment","Deployments",true,"sitepod-deployment-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" ReplicaSetClient(ext_api.ReplicaSet,ext_api.ReplicaSetList,"ReplicaSet","ReplicaSets",true,"sitepod-rs-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" SystemUserClient(v1.SystemUser,v1.SystemUserList,"SystemUser","SystemUsers",true,"systemuser-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" ConfigMapClient(k8s_api.ConfigMap,k8s_api.ConfigMapList,"ConfigMap","ConfigMaps",true,"sitepod-cm-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" ClusterClient(v1.Cluster,v1.ClusterList,"Cluster","Clusters",true,"sitepod-cluster-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" AppCompClient(v1.Appcomponent,v1.AppcomponentList,"AppComponent","AppComponents",true,"sitepod-appcomp-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" PodTaskClient(v1.Podtask,v1.PodtaskList,"PodTask","PodTasks",true,"sitepod-podtask-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" ServiceClient(k8s_api.Service,k8s_api.ServiceList,"Service","Services",true,"sitepod-service-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" WebsiteClient(v1.Website,v1.WebsiteList,"Website","Websites",true,"sitepod-website-")
