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
	resyncPeriodSitepodClient = 5 * time.Minute
)

func HackImportIgnoredSitepodClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeSitepodClient []int

type SitepodClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewSitepodClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *SitepodClient {
	c := &SitepodClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&v1.Sitepod{}),
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
		api.NewListWatchFromClient(c.rc, "Sitepods", c.ns, nil, pc),
		&v1.Sitepod{},
		resyncPeriodSitepodClient,
		indexers,
	)

	return c
}

func (c *SitepodClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *SitepodClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "Sitepod"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *SitepodClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

type ItemDefaultableSitepodClient interface {
	SetDefaults()
}

func (c *SitepodClient) NewEmpty() *v1.Sitepod {
	item := &v1.Sitepod{}
	item.GenerateName = "sitepod-"
	var aitem interface{}
	aitem = item
	if ditem, ok := aitem.(ItemDefaultableSitepodClient); ok {
		ditem.SetDefaults()
	}

	return item
}

//TODO: wrong location? shared?
func (c *SitepodClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

func (c *SitepodClient) UIDOf(obj interface{}) (string, bool) {

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", false
	}
	return string(accessor.GetUID()), true
}

//TODO: wrong location? shared?
func (c *SitepodClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *SitepodClient) MaybeGetByKey(key string) (*v1.Sitepod, bool) {

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
		glog.Infof("Got %s from informer store with rv %s", "Sitepod", item.ResourceVersion)
		return item, exists
	}
}

func (c *SitepodClient) GetByKey(key string) *v1.Sitepod {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found " + "Sitepod" + ": " + key)
	}

	return item
}

func (c *SitepodClient) ByIndexByKey(index string, key string) []*v1.Sitepod {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*v1.Sitepod{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *SitepodClient) BySitepodKey(sitepodKey string) []*v1.Sitepod {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *SitepodClient) BySitepodKeyFunc() func(string) []interface{} {
	return func(sitepodKey string) []interface{} {
		iArray := []interface{}{}
		for _, r := range c.ByIndexByKey("sitepod", sitepodKey) {
			iArray = append(iArray, r)
		}
		return iArray
	}
}

func (c *SitepodClient) MaybeSingleByUID(uid string) (*v1.Sitepod, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *SitepodClient) SingleBySitepodKey(sitepodKey string) *v1.Sitepod {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *SitepodClient) MaybeSingleBySitepodKey(sitepodKey string) (*v1.Sitepod, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "Sitepods", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *SitepodClient) Add(target *v1.Sitepod) *v1.Sitepod {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("Sitepods").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*v1.Sitepod)
	glog.Infof("Added %s - %s (rv: %s)", "Sitepod", item.Name, item.ResourceVersion)
	return item
}

func (c *SitepodClient) CloneItem(orig interface{}) *v1.Sitepod {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*v1.Sitepod)
}

func (c *SitepodClient) Update(target *v1.Sitepod) *v1.Sitepod {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("Sitepods").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*v1.Sitepod)
	return item
}

func (c *SitepodClient) UpdateOrAdd(target *v1.Sitepod) *v1.Sitepod {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *SitepodClient) FetchList(s labels.Selector) []*v1.Sitepod {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("Sitepods").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("Sitepods").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*v1.Sitepod{}
	kList := rObj.(*v1.SitepodList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *SitepodClient) TryDelete(target *v1.Sitepod) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("Sitepods").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("Sitepods").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *SitepodClient) Delete(target *v1.Sitepod) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *SitepodClient) DeleteFunc() func(interface{}) {
	return func(iTarget interface{}) {

		target := iTarget.(*v1.Sitepod)

		err := c.TryDelete(target)

		if err != nil {
			panic(err)
		}
	}
}

func (c *SitepodClient) List() []*v1.Sitepod {
	kItems := c.informer.GetStore().List()
	target := []*v1.Sitepod{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*v1.Sitepod))
	}
	return target
}

func (c *SitepodClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *SitepodClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
