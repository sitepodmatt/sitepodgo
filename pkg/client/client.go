package client

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	k8s_v1 "k8s.io/kubernetes/pkg/api/v1"
	ext_v1 "k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/runtime/serializer"
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
}

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" SitepodClient(v1.Sitepod,"Sitepod","Sitepods")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" PVClaimClient(k8s_api.PersistentVolumeClaim,"PersistentVolumeClaim","PersistentVolumeClaims")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" PVClient(k8s_api.PersistentVolume,"PersistentVolume","PersistentVolumes")

//go:generate gotemplate "sitepod.io/sitepod/pkg/client/clienttmpl" DeploymentClient(ext_api.Deployment,"Deployment","Deployments")

func NewClient(scheme *runtime.Scheme) *Client {
	client := &Client{scheme: scheme}
	client.serializer = serializer.NewCodecFactory(scheme)

	sitepodGroupVersion := &unversioned.GroupVersion{"stable.sitepod.io", "v1"}
	client.sitepodRestClient = client.buildRestClient("apis", sitepodGroupVersion)
	client.k8sCoreRestClient = client.buildRestClient("api", &k8s_v1.SchemeGroupVersion)
	client.k8sExtRestClient = client.buildRestClient("apis", &ext_v1.SchemeGroupVersion)

	return client
}

func (c *Client) Sitepods() *SitepodClient {
	return NewSitepodClient(c.sitepodRestClient, namespace)
}

func (c *Client) PVClaims() *PVClaimClient {
	return NewPVClaimClient(c.k8sCoreRestClient, namespace)
}

func (c *Client) PVs() *PVClient {
	return NewPVClient(c.k8sCoreRestClient, namespace)
}

func (c *Client) Deployments() *DeploymentClient {
	return NewDeploymentClient(c.k8sCoreRestClient, namespace)
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
