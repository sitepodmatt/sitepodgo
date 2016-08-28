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
	resyncPeriodPodClient = 5 * time.Minute
)

func HackImportIgnoredPodClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypePodClient []int

type PodClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewPodClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *PodClient {
	c := &PodClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&k8s_api.Pod{}),
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
		api.NewListWatchFromClient(c.rc, "Pods", c.ns, nil, pc),
		&k8s_api.Pod{},
		resyncPeriodPodClient,
		indexers,
	)

	return c
}

func (c *PodClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *PodClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "Pod"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *PodClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

type ItemDefaultablePodClient interface {
	SetDefaults()
}

func (c *PodClient) NewEmpty() *k8s_api.Pod {
	item := &k8s_api.Pod{}
	item.GenerateName = "sitepod-pod-"
	var aitem interface{}
	aitem = item
	if ditem, ok := aitem.(ItemDefaultablePodClient); ok {
		ditem.SetDefaults()
	}

	return item
}

//TODO: wrong location? shared?
func (c *PodClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

func (c *PodClient) UIDOf(obj interface{}) (string, bool) {

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", false
	}
	return string(accessor.GetUID()), true
}

//TODO: wrong location? shared?
func (c *PodClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *PodClient) MaybeGetByKey(key string) (*k8s_api.Pod, bool) {

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
		glog.Infof("Got %s from informer store with rv %s", "Pod", item.ResourceVersion)
		return item, exists
	}
}

func (c *PodClient) GetByKey(key string) *k8s_api.Pod {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found " + "Pod" + ": " + key)
	}

	return item
}

func (c *PodClient) ByIndexByKey(index string, key string) []*k8s_api.Pod {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*k8s_api.Pod{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *PodClient) BySitepodKey(sitepodKey string) []*k8s_api.Pod {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *PodClient) BySitepodKeyFunc() func(string) []interface{} {
	return func(sitepodKey string) []interface{} {
		iArray := []interface{}{}
		for _, r := range c.ByIndexByKey("sitepod", sitepodKey) {
			iArray = append(iArray, r)
		}
		return iArray
	}
}

func (c *PodClient) MaybeSingleByUID(uid string) (*k8s_api.Pod, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *PodClient) SingleBySitepodKey(sitepodKey string) *k8s_api.Pod {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *PodClient) MaybeSingleBySitepodKey(sitepodKey string) (*k8s_api.Pod, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "Pods", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *PodClient) Add(target *k8s_api.Pod) *k8s_api.Pod {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("Pods").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*k8s_api.Pod)
	glog.Infof("Added %s - %s (rv: %s)", "Pod", item.Name, item.ResourceVersion)
	return item
}

func (c *PodClient) CloneItem(orig interface{}) *k8s_api.Pod {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*k8s_api.Pod)
}

func (c *PodClient) Update(target *k8s_api.Pod) *k8s_api.Pod {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("Pods").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*k8s_api.Pod)
	return item
}

func (c *PodClient) UpdateOrAdd(target *k8s_api.Pod) *k8s_api.Pod {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *PodClient) FetchList(s labels.Selector) []*k8s_api.Pod {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("Pods").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("Pods").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*k8s_api.Pod{}
	kList := rObj.(*k8s_api.PodList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *PodClient) TryDelete(target *k8s_api.Pod) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("Pods").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("Pods").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *PodClient) Delete(target *k8s_api.Pod) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *PodClient) DeleteFunc() func(interface{}) {
	return func(iTarget interface{}) {

		target := iTarget.(*k8s_api.Pod)

		err := c.TryDelete(target)

		if err != nil {
			panic(err)
		}
	}
}

func (c *PodClient) List() []*k8s_api.Pod {
	kItems := c.informer.GetStore().List()
	target := []*k8s_api.Pod{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*k8s_api.Pod))
	}
	return target
}

func (c *PodClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *PodClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
