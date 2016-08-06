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
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"reflect"
	"sitepod.io/sitepod/pkg/api"
	"sitepod.io/sitepod/pkg/api/v1"
	"strings"
	"time"
)

var (
	resyncPeriodConfigMapClient = 5 * time.Minute
)

func HackImportIgnoredConfigMapClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeConfigMapClient []int

type ConfigMapClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewConfigMapClient(rc *restclient.RESTClient, ns string) *ConfigMapClient {
	c := &ConfigMapClient{
		rc:            rc,
		supportedType: reflect.TypeOf(&k8s_api.ConfigMap{}),
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
		accessor, _ := meta.Accessor(obj)
		return []string{string(accessor.GetUID())}, nil
	}

	c.informer = framework.NewSharedIndexInformer(
		api.NewListWatchFromClient(c.rc, "ConfigMaps", c.ns, nil, pc),
		&k8s_api.ConfigMap{},
		resyncPeriodConfigMapClient,
		indexers,
	)

	return c
}

func (c *ConfigMapClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *ConfigMapClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "ConfigMap"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *ConfigMapClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *ConfigMapClient) NewEmpty() *k8s_api.ConfigMap {
	item := &k8s_api.ConfigMap{}
	item.GenerateName = "sitepod-cm-"
	return item
}

//TODO: wrong location? shared?
func (c *ConfigMapClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

//TODO: wrong location? shared?
func (c *ConfigMapClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *ConfigMapClient) MaybeGetByKey(key string) (*k8s_api.ConfigMap, bool) {

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
		item := iObj.(*k8s_api.ConfigMap)
		glog.Infof("Got %s from informer store with rv %s", "ConfigMap", item.ResourceVersion)
		return item, exists
	}
}

func (c *ConfigMapClient) GetByKey(key string) *k8s_api.ConfigMap {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *ConfigMapClient) ByIndexByKey(index string, key string) []*k8s_api.ConfigMap {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*k8s_api.ConfigMap{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*k8s_api.ConfigMap))
	}
	return typedItems
}

func (c *ConfigMapClient) BySitepodKey(sitepodKey string) []*k8s_api.ConfigMap {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *ConfigMapClient) MaybeSingleByUID(uid string) (*k8s_api.ConfigMap, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *ConfigMapClient) SingleBySitepodKey(sitepodKey string) *k8s_api.ConfigMap {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *ConfigMapClient) MaybeSingleBySitepodKey(sitepodKey string) (*k8s_api.ConfigMap, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "ConfigMaps", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *ConfigMapClient) Add(target *k8s_api.ConfigMap) *k8s_api.ConfigMap {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("ConfigMaps").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*k8s_api.ConfigMap)
	glog.Infof("Added %s - %s (rv: %s)", "ConfigMap", item.Name, item.ResourceVersion)
	return item
}

func (c *ConfigMapClient) Update(target *k8s_api.ConfigMap) *k8s_api.ConfigMap {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("ConfigMaps").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*k8s_api.ConfigMap)
	return item
}

func (c *ConfigMapClient) UpdateOrAdd(target *k8s_api.ConfigMap) *k8s_api.ConfigMap {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *ConfigMapClient) FetchList(s labels.Selector) []*k8s_api.ConfigMap {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("ConfigMaps").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("ConfigMaps").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*k8s_api.ConfigMap{}
	kList := rObj.(*k8s_api.ConfigMapList)
	for _, kItem := range kList.Items {
		target = append(target, &kItem)
	}

	return target
}

func (c *ConfigMapClient) TryDelete(target *k8s_api.ConfigMap) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("ConfigMaps").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("ConfigMaps").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *ConfigMapClient) Delete(target *k8s_api.ConfigMap) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *ConfigMapClient) List() []*k8s_api.ConfigMap {
	kItems := c.informer.GetStore().List()
	target := []*k8s_api.ConfigMap{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*k8s_api.ConfigMap))
	}
	return target
}
