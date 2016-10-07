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
	resyncPeriodSitepodUserClient = 5 * time.Minute
)

func HackImportIgnoredSitepodUserClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeSitepodUserClient []int

type SitepodUserClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewSitepodUserClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *SitepodUserClient {
	c := &SitepodUserClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&v1.SitepodUser{}),
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
		api.NewListWatchFromClient(c.rc, "SitepodUsers", c.ns, nil, pc),
		&v1.SitepodUser{},
		resyncPeriodSitepodUserClient,
		indexers,
	)

	return c
}

func (c *SitepodUserClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *SitepodUserClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "SitepodUser"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *SitepodUserClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

type ItemDefaultableSitepodUserClient interface {
	SetDefaults()
}

func (c *SitepodUserClient) NewEmpty() *v1.SitepodUser {
	item := &v1.SitepodUser{}
	item.GenerateName = "sitepod-user-"
	var aitem interface{}
	aitem = item
	if ditem, ok := aitem.(ItemDefaultableSitepodUserClient); ok {
		ditem.SetDefaults()
	}

	return item
}

//TODO: wrong location? shared?
func (c *SitepodUserClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

func (c *SitepodUserClient) UIDOf(obj interface{}) (string, bool) {

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", false
	}
	return string(accessor.GetUID()), true
}

//TODO: wrong location? shared?
func (c *SitepodUserClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *SitepodUserClient) MaybeGetByKey(key string) (*v1.SitepodUser, bool) {

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
		glog.Infof("Got %s from informer store with rv %s", "SitepodUser", item.ResourceVersion)
		return item, exists
	}
}

func (c *SitepodUserClient) GetByKey(key string) *v1.SitepodUser {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found " + "SitepodUser" + ": " + key)
	}

	return item
}

func (c *SitepodUserClient) ByIndexByKey(index string, key string) []*v1.SitepodUser {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*v1.SitepodUser{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *SitepodUserClient) BySitepodKey(sitepodKey string) []*v1.SitepodUser {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *SitepodUserClient) BySitepodKeyFunc() func(string) []interface{} {
	return func(sitepodKey string) []interface{} {
		iArray := []interface{}{}
		for _, r := range c.ByIndexByKey("sitepod", sitepodKey) {
			iArray = append(iArray, r)
		}
		return iArray
	}
}

func (c *SitepodUserClient) MaybeSingleByUID(uid string) (*v1.SitepodUser, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *SitepodUserClient) SingleBySitepodKey(sitepodKey string) *v1.SitepodUser {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *SitepodUserClient) MaybeSingleBySitepodKey(sitepodKey string) (*v1.SitepodUser, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "SitepodUsers", sitepodKey, len(items))
		}

		return items[0], true
	}

}

type BeforeAdderSitepodUserClient interface {
	BeforeAdd()
}

func (c *SitepodUserClient) Add(target *v1.SitepodUser) *v1.SitepodUser {

	var itarget interface{}
	itarget = target
	if subject, ok := itarget.(BeforeAdderSitepodUserClient); ok {
		subject.BeforeAdd()
	}

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("SitepodUsers").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*v1.SitepodUser)
	glog.Infof("Added %s - %s (rv: %s)", "SitepodUser", item.Name, item.ResourceVersion)
	return item
}

func (c *SitepodUserClient) CloneItem(orig interface{}) *v1.SitepodUser {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*v1.SitepodUser)
}

func (c *SitepodUserClient) Update(target *v1.SitepodUser) *v1.SitepodUser {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("SitepodUsers").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*v1.SitepodUser)
	return item
}

func (c *SitepodUserClient) UpdateOrAdd(target *v1.SitepodUser) *v1.SitepodUser {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *SitepodUserClient) FetchList(s labels.Selector) []*v1.SitepodUser {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("SitepodUsers").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("SitepodUsers").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*v1.SitepodUser{}
	kList := rObj.(*v1.SitepodUserList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *SitepodUserClient) TryDelete(target *v1.SitepodUser) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("SitepodUsers").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("SitepodUsers").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *SitepodUserClient) Delete(target *v1.SitepodUser) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *SitepodUserClient) DeleteFunc() func(interface{}) {
	return func(iTarget interface{}) {

		target := iTarget.(*v1.SitepodUser)

		err := c.TryDelete(target)

		if err != nil {
			panic(err)
		}
	}
}

func (c *SitepodUserClient) List() []*v1.SitepodUser {
	kItems := c.informer.GetStore().List()
	target := []*v1.SitepodUser{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*v1.SitepodUser))
	}
	return target
}

func (c *SitepodUserClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *SitepodUserClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
