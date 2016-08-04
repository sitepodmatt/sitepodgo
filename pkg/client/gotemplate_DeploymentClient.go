package client

import (
	"errors"
	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/meta"
	ext_api "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"reflect"
	"sitepod.io/sitepod/pkg/api"
	"sitepod.io/sitepod/pkg/api/v1"
	"time"
)

var (
	resyncPeriodDeploymentClient = 5 * time.Minute
)

func HackImportIgnoredDeploymentClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName)

type DeploymentClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewDeploymentClient(rc *restclient.RESTClient, ns string) *DeploymentClient {
	return &DeploymentClient{
		rc:            rc,
		ns:            ns,
		supportedType: reflect.TypeOf(&ext_api.Deployment{}),
	}
}

func (c *DeploymentClient) StartInformer(stopCh <-chan struct{}) {
	//TODO do we still need to do this now we have single scheme
	pc := runtime.NewParameterCodec(k8s_api.Scheme)

	c.informer = framework.NewSharedIndexInformer(
		api.NewListWatchFromClient(c.rc, "x", c.ns, nil, pc),
		&ext_api.Deployment{},
		resyncPeriodDeploymentClient,
		nil,
	)

	c.informer.Run(stopCh)
}

func (c *DeploymentClient) NewEmpty() *ext_api.Deployment {
	return &ext_api.Deployment{}
}

func (c *DeploymentClient) MaybeGetByKey(key string) (*ext_api.Deployment, bool) {
	iObj, exists, err := c.informer.GetStore().GetByKey(key)

	if err != nil {
		panic(err)
	}

	if iObj == nil {
		return nil, exists
	} else {
		return iObj.(*ext_api.Deployment), exists
	}
}

func (c *DeploymentClient) GetByKey(key string) *ext_api.Deployment {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *DeploymentClient) BySitepodKey(sitepodKey string) ([]*ext_api.Deployment, error) {
	items, err := c.informer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		return nil, err
	}

	typedItems := []*ext_api.Deployment{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*ext_api.Deployment))
	}
	return typedItems, nil
}

func (c *DeploymentClient) SingleBySitepodKey(sitepodKey string) *ext_api.Deployment {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *DeploymentClient) MaybeSingleBySitepodKey(sitepodKey string) (*ext_api.Deployment, bool) {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}

}

func (c *DeploymentClient) Add(target *ext_api.Deployment) *ext_api.Deployment {

	result := c.rc.Post().Resource("Deployment").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}

	return r.(*ext_api.Deployment)
}

func (c *DeploymentClient) UpdateOrAdd(target *ext_api.Deployment) *ext_api.Deployment {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}

	uid := accessor.GetUID()
	if len(string(uid)) > 0 {
		rName := accessor.GetName()
		replacementTarget, err := c.rc.Put().Resource("Deployment").Name(rName).Body(target).Do().Get()
		if err != nil {
			panic(err)
		}
		return replacementTarget.(*ext_api.Deployment)
	} else {
		return c.Add(target)
	}
}
