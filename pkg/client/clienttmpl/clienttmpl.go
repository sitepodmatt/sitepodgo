package clienttmpl

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
	resyncPeriod = 5 * time.Minute
)

func HackImportIgnored(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName)
type ResourceType int

const ResourceName = "HolderName"

const ResourcePluralName = "HolderName"

type ClientTmpl struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewClientTmpl(rc *restclient.RESTClient, ns string) *ClientTmpl {
	return &ClientTmpl{
		rc:            rc,
		ns:            ns,
		supportedType: reflect.TypeOf(&ResourceType{}),
	}
}

func (c *ClientTmpl) StartInformer(stopCh <-chan struct{}) {
	//TODO do we still need to do this now we have single scheme
	pc := runtime.NewParameterCodec(k8s_api.Scheme)

	c.informer = framework.NewSharedIndexInformer(
		api.NewListWatchFromClient(c.rc, "x", c.ns, nil, pc),
		&ResourceType{},
		resyncPeriod,
		nil,
	)

	c.informer.Run(stopCh)
}

func (c *ClientTmpl) NewEmpty() *ResourceType {
	return &ResourceType{}
}

func (c *ClientTmpl) MaybeGetByKey(key string) (*ResourceType, bool) {
	iObj, exists, err := c.informer.GetStore().GetByKey(key)

	if err != nil {
		panic(err)
	}

	if iObj == nil {
		return nil, exists
	} else {
		return iObj.(*ResourceType), exists
	}
}

func (c *ClientTmpl) GetByKey(key string) *ResourceType {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *ClientTmpl) BySitepodKey(sitepodKey string) ([]*ResourceType, error) {
	items, err := c.informer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		return nil, err
	}

	typedItems := []*ResourceType{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*ResourceType))
	}
	return typedItems, nil
}

func (c *ClientTmpl) SingleBySitepodKey(sitepodKey string) *ResourceType {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *ClientTmpl) MaybeSingleBySitepodKey(sitepodKey string) (*ResourceType, bool) {

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

func (c *ClientTmpl) Add(target *ResourceType) *ResourceType {

	result := c.rc.Post().Resource(ResourceName).Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}

	return r.(*ResourceType)
}

func (c *ClientTmpl) UpdateOrAdd(target *ResourceType) *ResourceType {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}

	uid := accessor.GetUID()
	if len(string(uid)) > 0 {
		rName := accessor.GetName()
		replacementTarget, err := c.rc.Put().Resource(ResourceName).Name(rName).Body(target).Do().Get()
		if err != nil {
			panic(err)
		}
		return replacementTarget.(*ResourceType)
	} else {
		return c.Add(target)
	}
}
