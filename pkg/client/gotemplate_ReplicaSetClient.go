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
	resyncPeriodReplicaSetClient = 5 * time.Minute
)

func HackImportIgnoredReplicaSetClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeReplicaSetClient []int

type ReplicaSetClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewReplicaSetClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *ReplicaSetClient {
	c := &ReplicaSetClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&ext_api.ReplicaSet{}),
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
		api.NewListWatchFromClient(c.rc, "ReplicaSets", c.ns, nil, pc),
		&ext_api.ReplicaSet{},
		resyncPeriodReplicaSetClient,
		indexers,
	)

	return c
}

func (c *ReplicaSetClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *ReplicaSetClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "ReplicaSet"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *ReplicaSetClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

type ItemDefaultableReplicaSetClient interface {
	SetDefaults()
}

func (c *ReplicaSetClient) NewEmpty() *ext_api.ReplicaSet {
	item := &ext_api.ReplicaSet{}
	item.GenerateName = "sitepod-rs-"
	var aitem interface{}
	aitem = item
	if ditem, ok := aitem.(ItemDefaultableReplicaSetClient); ok {
		ditem.SetDefaults()
	}

	return item
}

//TODO: wrong location? shared?
func (c *ReplicaSetClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

func (c *ReplicaSetClient) UIDOf(obj interface{}) (string, bool) {

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", false
	}
	return string(accessor.GetUID()), true
}

//TODO: wrong location? shared?
func (c *ReplicaSetClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *ReplicaSetClient) MaybeGetByKey(key string) (*ext_api.ReplicaSet, bool) {

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
		glog.Infof("Got %s from informer store with rv %s", "ReplicaSet", item.ResourceVersion)
		return item, exists
	}
}

func (c *ReplicaSetClient) GetByKey(key string) *ext_api.ReplicaSet {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *ReplicaSetClient) ByIndexByKey(index string, key string) []*ext_api.ReplicaSet {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*ext_api.ReplicaSet{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *ReplicaSetClient) BySitepodKey(sitepodKey string) []*ext_api.ReplicaSet {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *ReplicaSetClient) BySitepodKeyFunc() func(string) []interface{} {
	return func(sitepodKey string) []interface{} {
		iArray := []interface{}{}
		for _, r := range c.ByIndexByKey("sitepod", sitepodKey) {
			iArray = append(iArray, r)
		}
		return iArray
	}
}

func (c *ReplicaSetClient) MaybeSingleByUID(uid string) (*ext_api.ReplicaSet, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *ReplicaSetClient) SingleBySitepodKey(sitepodKey string) *ext_api.ReplicaSet {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *ReplicaSetClient) MaybeSingleBySitepodKey(sitepodKey string) (*ext_api.ReplicaSet, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "ReplicaSets", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *ReplicaSetClient) Add(target *ext_api.ReplicaSet) *ext_api.ReplicaSet {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("ReplicaSets").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*ext_api.ReplicaSet)
	glog.Infof("Added %s - %s (rv: %s)", "ReplicaSet", item.Name, item.ResourceVersion)
	return item
}

func (c *ReplicaSetClient) CloneItem(orig interface{}) *ext_api.ReplicaSet {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*ext_api.ReplicaSet)
}

func (c *ReplicaSetClient) Update(target *ext_api.ReplicaSet) *ext_api.ReplicaSet {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("ReplicaSets").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*ext_api.ReplicaSet)
	return item
}

func (c *ReplicaSetClient) UpdateOrAdd(target *ext_api.ReplicaSet) *ext_api.ReplicaSet {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *ReplicaSetClient) FetchList(s labels.Selector) []*ext_api.ReplicaSet {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("ReplicaSets").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("ReplicaSets").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*ext_api.ReplicaSet{}
	kList := rObj.(*ext_api.ReplicaSetList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *ReplicaSetClient) TryDelete(target *ext_api.ReplicaSet) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("ReplicaSets").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("ReplicaSets").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *ReplicaSetClient) Delete(target *ext_api.ReplicaSet) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *ReplicaSetClient) DeleteFunc() func(interface{}) {
	return func(iTarget interface{}) {

		target := iTarget.(*ext_api.ReplicaSet)

		err := c.TryDelete(target)

		if err != nil {
			panic(err)
		}
	}
}

func (c *ReplicaSetClient) List() []*ext_api.ReplicaSet {
	kItems := c.informer.GetStore().List()
	target := []*ext_api.ReplicaSet{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*ext_api.ReplicaSet))
	}
	return target
}

func (c *ReplicaSetClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *ReplicaSetClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
