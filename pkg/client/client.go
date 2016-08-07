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

//TODO: inject as host configuration
var (
	hostPath  = "http://127.0.0.1:8080"
	namespace = "default"
)

type Client struct {
	scheme            *runtime.Scheme
	serializer        runtime.NegotiatedSerializer
	sitepodRestClient *restclient.RESTClient
	k8sCoreRestClient *restclient.RESTClient
	k8sExtRestClient  *restclient.RESTClient
	cachedClients     map[string]interface{}
	cachedClientMutex sync.Mutex
}

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" SitepodClient(v1.Sitepod,v1.SitepodList,"Sitepod","Sitepods",true,"sitepod-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" PVClaimClient(k8s_api.PersistentVolumeClaim,k8s_api.PersistentVolumeClaimList,"PersistentVolumeClaim","PersistentVolumeClaims",true,"sitepod-pvc-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" PVClient(k8s_api.PersistentVolume,k8s_api.PersistentVolumeList,"PersistentVolume","PersistentVolumes",false,"sitepod-pv-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" DeploymentClient(ext_api.Deployment,ext_api.DeploymentList,"Deployment","Deployments",true,"sitepod-deployment-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" ReplicaSetClient(ext_api.ReplicaSet,ext_api.ReplicaSetList,"ReplicaSet","ReplicaSets",true,"sitepod-rs-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" SystemUserClient(v1.SystemUser,v1.SystemUserList,"SystemUser","SystemUsers",true,"systemuser-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" ConfigMapClient(k8s_api.ConfigMap,k8s_api.ConfigMapList,"ConfigMap","ConfigMaps",true,"sitepod-cm-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" ClusterClient(v1.Cluster,v1.ClusterList,"Cluster","Clusters",true,"sitepod-cluster-")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" AppCompClient(v1.AppComponent,v1.AppComponentList,"AppComponent","AppComponents",true,"sitepod-appcomp-")

func NewClient(scheme *runtime.Scheme) *Client {
	client := &Client{scheme: scheme}
	client.cachedClients = make(map[string]interface{})
	client.serializer = serializer.NewCodecFactory(scheme)

	sitepodGroupVersion := &unversioned.GroupVersion{"stable.sitepod.io", "v1"}
	client.sitepodRestClient = client.buildRestClient("apis", sitepodGroupVersion)
	client.k8sCoreRestClient = client.buildRestClient("api", &k8s_v1.SchemeGroupVersion)
	client.k8sExtRestClient = client.buildRestClient("apis", &ext_v1.SchemeGroupVersion)

	return client
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
	return c.usingCache("sitepods", func() interface{} { return NewSitepodClient(c.sitepodRestClient, namespace) }).(*SitepodClient)
}

func (c *Client) PVClaims() *PVClaimClient {
	return c.usingCache("pvclaims", func() interface{} { return NewPVClaimClient(c.k8sCoreRestClient, namespace) }).(*PVClaimClient)
}

func (c *Client) PVs() *PVClient {
	return c.usingCache("pvs", func() interface{} { return NewPVClient(c.k8sCoreRestClient, namespace) }).(*PVClient)
}

func (c *Client) Deployments() *DeploymentClient {
	return c.usingCache("deployments", func() interface{} { return NewDeploymentClient(c.k8sExtRestClient, namespace) }).(*DeploymentClient)
}

func (c *Client) ReplicaSets() *ReplicaSetClient {
	return c.usingCache("replicasets", func() interface{} { return NewReplicaSetClient(c.k8sExtRestClient, namespace) }).(*ReplicaSetClient)
}

func (c *Client) SystemUsers() *SystemUserClient {
	return c.usingCache("systemusers", func() interface{} { return NewSystemUserClient(c.sitepodRestClient, namespace) }).(*SystemUserClient)
}

func (c *Client) ConfigMaps() *ConfigMapClient {
	return c.usingCache("configmaps", func() interface{} { return NewConfigMapClient(c.k8sCoreRestClient, namespace) }).(*ConfigMapClient)
}

func (c *Client) Clusters() *ClusterClient {
	return c.usingCache("clusters", func() interface{} { return NewClusterClient(c.sitepodRestClient, namespace) }).(*ClusterClient)
}

func (c *Client) AppComps() *AppCompClient {
	return c.usingCache("appcomps", func() interface{} { return NewAppCompClient(c.sitepodRestClient, namespace) }).(*AppCompClient)
}

func (c *Client) buildRestClient(apiPath string, gv *unversioned.GroupVersion) *restclient.RESTClient {

	rcConfig := &restclient.Config{
		Host:    hostPath,
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

	return rc
}
