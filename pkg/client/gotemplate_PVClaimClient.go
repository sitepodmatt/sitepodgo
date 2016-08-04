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
	resyncPeriodPVClaimClient = 5 * time.Minute
)

func HackImportIgnoredPVClaimClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName)

type PVClaimClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewPVClaimClient(rc *restclient.RESTClient, ns string) *PVClaimClient {
	return &PVClaimClient{
		rc:            rc,
		ns:            ns,
		supportedType: reflect.TypeOf(&k8s_api.PersistentVolumeClaim{}),
	}
}

func (c *PVClaimClient) StartInformer(stopCh <-chan struct{}) {
	//TODO do we still need to do this now we have single scheme
	pc := runtime.NewParameterCodec(k8s_api.Scheme)

	c.informer = framework.NewSharedIndexInformer(
		api.NewListWatchFromClient(c.rc, "x", c.ns, nil, pc),
		&k8s_api.PersistentVolumeClaim{},
		resyncPeriodPVClaimClient,
		nil,
	)

	c.informer.Run(stopCh)
}

func (c *PVClaimClient) NewEmpty() *k8s_api.PersistentVolumeClaim {
	return &k8s_api.PersistentVolumeClaim{}
}

func (c *PVClaimClient) MaybeGetByKey(key string) (*k8s_api.PersistentVolumeClaim, bool) {
	iObj, exists, err := c.informer.GetStore().GetByKey(key)

	if err != nil {
		panic(err)
	}

	if iObj == nil {
		return nil, exists
	} else {
		return iObj.(*k8s_api.PersistentVolumeClaim), exists
	}
}

func (c *PVClaimClient) GetByKey(key string) *k8s_api.PersistentVolumeClaim {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *PVClaimClient) BySitepodKey(sitepodKey string) ([]*k8s_api.PersistentVolumeClaim, error) {
	items, err := c.informer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		return nil, err
	}

	typedItems := []*k8s_api.PersistentVolumeClaim{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*k8s_api.PersistentVolumeClaim))
	}
	return typedItems, nil
}

func (c *PVClaimClient) SingleBySitepodKey(sitepodKey string) *k8s_api.PersistentVolumeClaim {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *PVClaimClient) MaybeSingleBySitepodKey(sitepodKey string) (*k8s_api.PersistentVolumeClaim, bool) {

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

func (c *PVClaimClient) Add(target *k8s_api.PersistentVolumeClaim) *k8s_api.PersistentVolumeClaim {

	result := c.rc.Post().Resource("PersistentVolumeClaim").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}

	return r.(*k8s_api.PersistentVolumeClaim)
}

func (c *PVClaimClient) UpdateOrAdd(target *k8s_api.PersistentVolumeClaim) *k8s_api.PersistentVolumeClaim {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}

	uid := accessor.GetUID()
	if len(string(uid)) > 0 {
		rName := accessor.GetName()
		replacementTarget, err := c.rc.Put().Resource("PersistentVolumeClaim").Name(rName).Body(target).Do().Get()
		if err != nil {
			panic(err)
		}
		return replacementTarget.(*k8s_api.PersistentVolumeClaim)
	} else {
		return c.Add(target)
	}
}
