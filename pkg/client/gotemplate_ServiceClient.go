package client

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
	"k8s.io/kubernetes/pkg/conversion"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"reflect"
	"sitepod.io/sitepod/pkg/api"
	"sitepod.io/sitepod/pkg/api/v1"
	"strings"
	"time"
)

var (
	resyncPeriodServiceClient = 5 * time.Minute
)

func HackImportIgnoredServiceClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeServiceClient []int

type ServiceClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewServiceClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *ServiceClient {
	c := &ServiceClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&k8s_api.Service{}),
	}

	if true {
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

	indexers["uid"] = func(obj interface{}) ([]string, error) {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			panic(err)
		}
		return []string{string(accessor.GetUID())}, nil
	}

	c.informer = framework.NewSharedIndexInformer(
		api.NewListWatchFromClient(c.rc, "Services", c.ns, nil, pc),
		&k8s_api.Service{},
		resyncPeriodServiceClient,
		indexers,
	)

	return c
}

func (c *ServiceClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *ServiceClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "Service"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *ServiceClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

type ItemDefaultableServiceClient interface {
	SetDefaults()
}

func (c *ServiceClient) NewEmpty() *k8s_api.Service {
	item := &k8s_api.Service{}
	item.GenerateName = "sitepod-service-"
	var aitem interface{}
	aitem = item
	if ditem, ok := aitem.(ItemDefaultableServiceClient); ok {
		ditem.SetDefaults()
	}

	return item
}

//TODO: wrong location? shared?
func (c *ServiceClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

func (c *ServiceClient) UIDOf(obj interface{}) (string, bool) {

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", false
	}
	return string(accessor.GetUID()), true
}

//TODO: wrong location? shared?
func (c *ServiceClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *ServiceClient) MaybeGetByKey(key string) (*k8s_api.Service, bool) {

	if !strings.Contains(key, "/") && true {
		key = fmt.Sprintf("%s/%s", c.ns, key)
	}

	iObj, exists, err := c.informer.GetStore().GetByKey(key)

	if err != nil {
		panic(err)
	}

	if iObj == nil {
		return nil, exists
	} else {
		item := c.CloneItem(iObj)
		glog.Infof("Got %s from informer store with rv %s", "Service", item.ResourceVersion)
		return item, exists
	}
}

func (c *ServiceClient) GetByKey(key string) *k8s_api.Service {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *ServiceClient) ByIndexByKey(index string, key string) []*k8s_api.Service {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*k8s_api.Service{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *ServiceClient) BySitepodKey(sitepodKey string) []*k8s_api.Service {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *ServiceClient) BySitepodKeyFunc() func(string) []interface{} {
	return func(sitepodKey string) []interface{} {
		iArray := []interface{}{}
		for _, r := range c.ByIndexByKey("sitepod", sitepodKey) {
			iArray = append(iArray, r)
		}
		return iArray
	}
}

func (c *ServiceClient) MaybeSingleByUID(uid string) (*k8s_api.Service, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *ServiceClient) SingleBySitepodKey(sitepodKey string) *k8s_api.Service {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *ServiceClient) MaybeSingleBySitepodKey(sitepodKey string) (*k8s_api.Service, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "Services", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *ServiceClient) Add(target *k8s_api.Service) *k8s_api.Service {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("Services").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*k8s_api.Service)
	glog.Infof("Added %s - %s (rv: %s)", "Service", item.Name, item.ResourceVersion)
	return item
}

func (c *ServiceClient) CloneItem(orig interface{}) *k8s_api.Service {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*k8s_api.Service)
}

func (c *ServiceClient) Update(target *k8s_api.Service) *k8s_api.Service {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("Services").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*k8s_api.Service)
	return item
}

func (c *ServiceClient) UpdateOrAdd(target *k8s_api.Service) *k8s_api.Service {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *ServiceClient) FetchList(s labels.Selector) []*k8s_api.Service {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("Services").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("Services").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*k8s_api.Service{}
	kList := rObj.(*k8s_api.ServiceList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *ServiceClient) TryDelete(target *k8s_api.Service) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("Services").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("Services").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *ServiceClient) Delete(target *k8s_api.Service) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *ServiceClient) DeleteFunc() func(interface{}) {
	return func(iTarget interface{}) {

		target := iTarget.(*k8s_api.Service)

		err := c.TryDelete(target)

		if err != nil {
			panic(err)
		}
	}
}

func (c *ServiceClient) List() []*k8s_api.Service {
	kItems := c.informer.GetStore().List()
	target := []*k8s_api.Service{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*k8s_api.Service))
	}
	return target
}

func (c *ServiceClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *ServiceClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
