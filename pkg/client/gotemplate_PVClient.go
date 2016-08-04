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
	resyncPeriodPVClient = 5 * time.Minute
)

func HackImportIgnoredPVClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName)

type PVClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewPVClient(rc *restclient.RESTClient, ns string) *PVClient {
	return &PVClient{
		rc:            rc,
		ns:            ns,
		supportedType: reflect.TypeOf(&k8s_api.PersistentVolume{}),
	}
}

func (c *PVClient) StartInformer(stopCh <-chan struct{}) {
	//TODO do we still need to do this now we have single scheme
	pc := runtime.NewParameterCodec(k8s_api.Scheme)

	c.informer = framework.NewSharedIndexInformer(
		api.NewListWatchFromClient(c.rc, "x", c.ns, nil, pc),
		&k8s_api.PersistentVolume{},
		resyncPeriodPVClient,
		nil,
	)

	c.informer.Run(stopCh)
}

func (c *PVClient) NewEmpty() *k8s_api.PersistentVolume {
	return &k8s_api.PersistentVolume{}
}

func (c *PVClient) MaybeGetByKey(key string) (*k8s_api.PersistentVolume, bool) {
	iObj, exists, err := c.informer.GetStore().GetByKey(key)

	if err != nil {
		panic(err)
	}

	if iObj == nil {
		return nil, exists
	} else {
		return iObj.(*k8s_api.PersistentVolume), exists
	}
}

func (c *PVClient) GetByKey(key string) *k8s_api.PersistentVolume {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *PVClient) BySitepodKey(sitepodKey string) ([]*k8s_api.PersistentVolume, error) {
	items, err := c.informer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		return nil, err
	}

	typedItems := []*k8s_api.PersistentVolume{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*k8s_api.PersistentVolume))
	}
	return typedItems, nil
}

func (c *PVClient) SingleBySitepodKey(sitepodKey string) *k8s_api.PersistentVolume {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *PVClient) MaybeSingleBySitepodKey(sitepodKey string) (*k8s_api.PersistentVolume, bool) {

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

func (c *PVClient) Add(target *k8s_api.PersistentVolume) *k8s_api.PersistentVolume {

	result := c.rc.Post().Resource("PersistentVolume").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}

	return r.(*k8s_api.PersistentVolume)
}

func (c *PVClient) UpdateOrAdd(target *k8s_api.PersistentVolume) *k8s_api.PersistentVolume {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}

	uid := accessor.GetUID()
	if len(string(uid)) > 0 {
		rName := accessor.GetName()
		replacementTarget, err := c.rc.Put().Resource("PersistentVolume").Name(rName).Body(target).Do().Get()
		if err != nil {
			panic(err)
		}
		return replacementTarget.(*k8s_api.PersistentVolume)
	} else {
		return c.Add(target)
	}
}
