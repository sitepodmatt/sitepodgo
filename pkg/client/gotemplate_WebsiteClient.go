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
	resyncPeriodWebsiteClient = 5 * time.Minute
)

func HackImportIgnoredWebsiteClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeWebsiteClient []int

type WebsiteClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewWebsiteClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *WebsiteClient {
	c := &WebsiteClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&v1.Website{}),
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
		api.NewListWatchFromClient(c.rc, "Websites", c.ns, nil, pc),
		&v1.Website{},
		resyncPeriodWebsiteClient,
		indexers,
	)

	return c
}

func (c *WebsiteClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *WebsiteClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "Website"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *WebsiteClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

type ItemDefaultableWebsiteClient interface {
	SetDefaults()
}

func (c *WebsiteClient) NewEmpty() *v1.Website {
	item := &v1.Website{}
	item.GenerateName = "sitepod-website-"
	var aitem interface{}
	aitem = item
	if ditem, ok := aitem.(ItemDefaultableWebsiteClient); ok {
		ditem.SetDefaults()
	}

	return item
}

//TODO: wrong location? shared?
func (c *WebsiteClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

func (c *WebsiteClient) UIDOf(obj interface{}) (string, bool) {

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", false
	}
	return string(accessor.GetUID()), true
}

//TODO: wrong location? shared?
func (c *WebsiteClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *WebsiteClient) MaybeGetByKey(key string) (*v1.Website, bool) {

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
		glog.Infof("Got %s from informer store with rv %s", "Website", item.ResourceVersion)
		return item, exists
	}
}

func (c *WebsiteClient) GetByKey(key string) *v1.Website {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found " + "Website" + ": " + key)
	}

	return item
}

func (c *WebsiteClient) ByIndexByKey(index string, key string) []*v1.Website {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*v1.Website{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *WebsiteClient) BySitepodKey(sitepodKey string) []*v1.Website {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *WebsiteClient) BySitepodKeyFunc() func(string) []interface{} {
	return func(sitepodKey string) []interface{} {
		iArray := []interface{}{}
		for _, r := range c.ByIndexByKey("sitepod", sitepodKey) {
			iArray = append(iArray, r)
		}
		return iArray
	}
}

func (c *WebsiteClient) MaybeSingleByUID(uid string) (*v1.Website, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *WebsiteClient) SingleBySitepodKey(sitepodKey string) *v1.Website {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *WebsiteClient) MaybeSingleBySitepodKey(sitepodKey string) (*v1.Website, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "Websites", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *WebsiteClient) Add(target *v1.Website) *v1.Website {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("Websites").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*v1.Website)
	glog.Infof("Added %s - %s (rv: %s)", "Website", item.Name, item.ResourceVersion)
	return item
}

func (c *WebsiteClient) CloneItem(orig interface{}) *v1.Website {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*v1.Website)
}

func (c *WebsiteClient) Update(target *v1.Website) *v1.Website {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("Websites").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*v1.Website)
	return item
}

func (c *WebsiteClient) UpdateOrAdd(target *v1.Website) *v1.Website {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *WebsiteClient) FetchList(s labels.Selector) []*v1.Website {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("Websites").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("Websites").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*v1.Website{}
	kList := rObj.(*v1.WebsiteList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *WebsiteClient) TryDelete(target *v1.Website) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("Websites").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("Websites").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *WebsiteClient) Delete(target *v1.Website) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *WebsiteClient) DeleteFunc() func(interface{}) {
	return func(iTarget interface{}) {

		target := iTarget.(*v1.Website)

		err := c.TryDelete(target)

		if err != nil {
			panic(err)
		}
	}
}

func (c *WebsiteClient) List() []*v1.Website {
	kItems := c.informer.GetStore().List()
	target := []*v1.Website{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*v1.Website))
	}
	return target
}

func (c *WebsiteClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *WebsiteClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
