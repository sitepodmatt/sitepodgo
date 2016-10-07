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
	resyncPeriodPVClient = 5 * time.Minute
)

func HackImportIgnoredPVClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypePVClient []int

type PVClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewPVClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *PVClient {
	c := &PVClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&k8s_api.PersistentVolume{}),
	}

	if false {
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
		api.NewListWatchFromClient(c.rc, "PersistentVolumes", c.ns, nil, pc),
		&k8s_api.PersistentVolume{},
		resyncPeriodPVClient,
		indexers,
	)

	return c
}

func (c *PVClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *PVClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "PersistentVolume"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *PVClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

type ItemDefaultablePVClient interface {
	SetDefaults()
}

func (c *PVClient) NewEmpty() *k8s_api.PersistentVolume {
	item := &k8s_api.PersistentVolume{}
	item.GenerateName = "sitepod-pv-"
	var aitem interface{}
	aitem = item
	if ditem, ok := aitem.(ItemDefaultablePVClient); ok {
		ditem.SetDefaults()
	}

	return item
}

//TODO: wrong location? shared?
func (c *PVClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

func (c *PVClient) UIDOf(obj interface{}) (string, bool) {

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", false
	}
	return string(accessor.GetUID()), true
}

//TODO: wrong location? shared?
func (c *PVClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *PVClient) MaybeGetByKey(key string) (*k8s_api.PersistentVolume, bool) {

	if !strings.Contains(key, "/") && false {
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
		glog.Infof("Got %s from informer store with rv %s", "PersistentVolume", item.ResourceVersion)
		return item, exists
	}
}

func (c *PVClient) GetByKey(key string) *k8s_api.PersistentVolume {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found " + "PersistentVolume" + ": " + key)
	}

	return item
}

func (c *PVClient) ByIndexByKey(index string, key string) []*k8s_api.PersistentVolume {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*k8s_api.PersistentVolume{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *PVClient) BySitepodKey(sitepodKey string) []*k8s_api.PersistentVolume {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *PVClient) BySitepodKeyFunc() func(string) []interface{} {
	return func(sitepodKey string) []interface{} {
		iArray := []interface{}{}
		for _, r := range c.ByIndexByKey("sitepod", sitepodKey) {
			iArray = append(iArray, r)
		}
		return iArray
	}
}

func (c *PVClient) MaybeSingleByUID(uid string) (*k8s_api.PersistentVolume, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *PVClient) SingleBySitepodKey(sitepodKey string) *k8s_api.PersistentVolume {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *PVClient) MaybeSingleBySitepodKey(sitepodKey string) (*k8s_api.PersistentVolume, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "PersistentVolumes", sitepodKey, len(items))
		}

		return items[0], true
	}

}

type BeforeAdderPVClient interface {
	BeforeAdd()
}

func (c *PVClient) Add(target *k8s_api.PersistentVolume) *k8s_api.PersistentVolume {

	var itarget interface{}
	itarget = target
	if subject, ok := itarget.(BeforeAdderPVClient); ok {
		subject.BeforeAdd()
	}

	rcReq := c.rc.Post()
	if false {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("PersistentVolumes").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*k8s_api.PersistentVolume)
	glog.Infof("Added %s - %s (rv: %s)", "PersistentVolume", item.Name, item.ResourceVersion)
	return item
}

func (c *PVClient) CloneItem(orig interface{}) *k8s_api.PersistentVolume {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*k8s_api.PersistentVolume)
}

func (c *PVClient) Update(target *k8s_api.PersistentVolume) *k8s_api.PersistentVolume {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if false {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("PersistentVolumes").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*k8s_api.PersistentVolume)
	return item
}

func (c *PVClient) UpdateOrAdd(target *k8s_api.PersistentVolume) *k8s_api.PersistentVolume {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *PVClient) FetchList(s labels.Selector) []*k8s_api.PersistentVolume {

	var prc *restclient.Request
	if !false {
		prc = c.rc.Get().Resource("PersistentVolumes").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("PersistentVolumes").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*k8s_api.PersistentVolume{}
	kList := rObj.(*k8s_api.PersistentVolumeList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *PVClient) TryDelete(target *k8s_api.PersistentVolume) error {

	var prc *restclient.Request
	if !false {
		prc = c.rc.Delete().Resource("PersistentVolumes").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("PersistentVolumes").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *PVClient) Delete(target *k8s_api.PersistentVolume) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *PVClient) DeleteFunc() func(interface{}) {
	return func(iTarget interface{}) {

		target := iTarget.(*k8s_api.PersistentVolume)

		err := c.TryDelete(target)

		if err != nil {
			panic(err)
		}
	}
}

func (c *PVClient) List() []*k8s_api.PersistentVolume {
	kItems := c.informer.GetStore().List()
	target := []*k8s_api.PersistentVolume{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*k8s_api.PersistentVolume))
	}
	return target
}

func (c *PVClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *PVClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
