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
	resyncPeriodSitepodClient = 5 * time.Minute
)

func HackImportIgnoredSitepodClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName)

type SitepodClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewSitepodClient(rc *restclient.RESTClient, ns string) *SitepodClient {
	return &SitepodClient{
		rc:            rc,
		ns:            ns,
		supportedType: reflect.TypeOf(&v1.Sitepod{}),
	}
}

func (c *SitepodClient) StartInformer(stopCh <-chan struct{}) {
	//TODO do we still need to do this now we have single scheme
	pc := runtime.NewParameterCodec(k8s_api.Scheme)

	c.informer = framework.NewSharedIndexInformer(
		api.NewListWatchFromClient(c.rc, "x", c.ns, nil, pc),
		&v1.Sitepod{},
		resyncPeriodSitepodClient,
		nil,
	)

	c.informer.Run(stopCh)
}

func (c *SitepodClient) NewEmpty() *v1.Sitepod {
	return &v1.Sitepod{}
}

func (c *SitepodClient) MaybeGetByKey(key string) (*v1.Sitepod, bool) {
	iObj, exists, err := c.informer.GetStore().GetByKey(key)

	if err != nil {
		panic(err)
	}

	if iObj == nil {
		return nil, exists
	} else {
		return iObj.(*v1.Sitepod), exists
	}
}

func (c *SitepodClient) GetByKey(key string) *v1.Sitepod {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *SitepodClient) BySitepodKey(sitepodKey string) ([]*v1.Sitepod, error) {
	items, err := c.informer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		return nil, err
	}

	typedItems := []*v1.Sitepod{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*v1.Sitepod))
	}
	return typedItems, nil
}

func (c *SitepodClient) SingleBySitepodKey(sitepodKey string) *v1.Sitepod {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *SitepodClient) MaybeSingleBySitepodKey(sitepodKey string) (*v1.Sitepod, bool) {

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

func (c *SitepodClient) Add(target *v1.Sitepod) *v1.Sitepod {

	result := c.rc.Post().Resource("Sitepod").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}

	return r.(*v1.Sitepod)
}

func (c *SitepodClient) UpdateOrAdd(target *v1.Sitepod) *v1.Sitepod {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}

	uid := accessor.GetUID()
	if len(string(uid)) > 0 {
		rName := accessor.GetName()
		replacementTarget, err := c.rc.Put().Resource("Sitepod").Name(rName).Body(target).Do().Get()
		if err != nil {
			panic(err)
		}
		return replacementTarget.(*v1.Sitepod)
	} else {
		return c.Add(target)
	}
}
