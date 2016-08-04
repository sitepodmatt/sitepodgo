package clienttmpl

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/meta"
	ext_api "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"reflect"
	"sitepod.io/sitepod/pkg/api"
	"sitepod.io/sitepod/pkg/api/v1"
	"strings"
	"time"
)

var (
	resyncPeriod = 5 * time.Minute
)

func HackImportIgnored(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)
type ResourceType int

const ResourceName = "HolderName"

const ResourcePluralName = "HolderName"

const Namespaced = true

const DefaultGenName = "sitepod-x-"

type ClientTmpl struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewClientTmpl(rc *restclient.RESTClient, ns string) *ClientTmpl {
	c := &ClientTmpl{
		rc:            rc,
		supportedType: reflect.TypeOf(&ResourceType{}),
	}

	if Namespaced {
		c.ns = ns
	}

	pc := runtime.NewParameterCodec(k8s_api.Scheme)

	indexers := make(cache.Indexers)
	indexers["sitepod"] = func(obj interface{}) ([]string, error) {
		accessor, _ := meta.Accessor(obj)
		labels := accessor.GetLabels()
		if _, ok := labels["sitepod"]; ok {
			return []string{labels["sitepod"]}, nil
		} else {
			return []string{}, nil
		}
	}
	c.informer = framework.NewSharedIndexInformer(
		api.NewListWatchFromClient(c.rc, ResourcePluralName, c.ns, nil, pc),
		&ResourceType{},
		resyncPeriod,
		indexers,
	)

	return c
}

func (c *ClientTmpl) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *ClientTmpl) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", ResourceName))
	}

	c.informer.AddEventHandler(reh)
}

func (c *ClientTmpl) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *ClientTmpl) NewEmpty() *ResourceType {
	item := &ResourceType{}
	item.GenerateName = DefaultGenName
	return item
}

//TODO: wrong location? shared?
func (c *ClientTmpl) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

//TODO: wrong location? shared?
func (c *ClientTmpl) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *ClientTmpl) MaybeGetByKey(key string) (*ResourceType, bool) {

	if !strings.Contains(key, "/") && Namespaced {
		key = fmt.Sprintf("%s/%s", c.ns, key)
	}

	iObj, exists, err := c.informer.GetStore().GetByKey(key)

	if err != nil {
		panic(err)
	}

	if iObj == nil {
		return nil, exists
	} else {
		item := iObj.(*ResourceType)
		glog.Infof("Got %s from informer store with rv %s", ResourceName, item.ResourceVersion)
		return item, exists
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

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", ResourcePluralName, sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *ClientTmpl) Add(target *ResourceType) *ResourceType {

	rcReq := c.rc.Post()
	if Namespaced {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource(ResourcePluralName).Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*ResourceType)
	glog.Infof("Added %s - %s (rv: %s)", ResourceName, item.Name, item.ResourceVersion)
	return item
}

func (c *ClientTmpl) UpdateOrAdd(target *ResourceType) *ResourceType {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}

	uid := accessor.GetUID()
	if len(string(uid)) > 0 {
		rName := accessor.GetName()
		rcReq := c.rc.Put()
		if Namespaced {
			rcReq = rcReq.Namespace(c.ns)
		}
		replacementTarget, err := rcReq.Resource(ResourcePluralName).Name(rName).Body(target).Do().Get()
		if err != nil {
			panic(err)
		}
		item := replacementTarget.(*ResourceType)
		glog.Infof("Updated %s - %s (rv: %s)", ResourceName, item.Name, item.ResourceVersion)
		return item
	} else {
		return c.Add(target)
	}
}
