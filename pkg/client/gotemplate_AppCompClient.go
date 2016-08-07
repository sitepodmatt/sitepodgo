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
	resyncPeriodAppCompClient = 5 * time.Minute
)

func HackImportIgnoredAppCompClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeAppCompClient []int

type AppCompClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewAppCompClient(rc *restclient.RESTClient, ns string) *AppCompClient {
	c := &AppCompClient{
		rc:            rc,
		supportedType: reflect.TypeOf(&v1.AppComponent{}),
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
		api.NewListWatchFromClient(c.rc, "AppComponents", c.ns, nil, pc),
		&v1.AppComponent{},
		resyncPeriodAppCompClient,
		indexers,
	)

	return c
}

func (c *AppCompClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *AppCompClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "AppComponent"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *AppCompClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *AppCompClient) NewEmpty() *v1.AppComponent {
	item := &v1.AppComponent{}
	item.GenerateName = "sitepod-appcomp-"
	return item
}

//TODO: wrong location? shared?
func (c *AppCompClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

//TODO: wrong location? shared?
func (c *AppCompClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *AppCompClient) MaybeGetByKey(key string) (*v1.AppComponent, bool) {

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
		item := iObj.(*v1.AppComponent)
		glog.Infof("Got %s from informer store with rv %s", "AppComponent", item.ResourceVersion)
		return item, exists
	}
}

func (c *AppCompClient) GetByKey(key string) *v1.AppComponent {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *AppCompClient) ByIndexByKey(index string, key string) []*v1.AppComponent {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*v1.AppComponent{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*v1.AppComponent))
	}
	return typedItems
}

func (c *AppCompClient) BySitepodKey(sitepodKey string) []*v1.AppComponent {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *AppCompClient) MaybeSingleByUID(uid string) (*v1.AppComponent, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *AppCompClient) SingleBySitepodKey(sitepodKey string) *v1.AppComponent {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *AppCompClient) MaybeSingleBySitepodKey(sitepodKey string) (*v1.AppComponent, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "AppComponents", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *AppCompClient) Add(target *v1.AppComponent) *v1.AppComponent {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("AppComponents").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*v1.AppComponent)
	glog.Infof("Added %s - %s (rv: %s)", "AppComponent", item.Name, item.ResourceVersion)
	return item
}

func (c *AppCompClient) Update(target *v1.AppComponent) *v1.AppComponent {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("AppComponents").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*v1.AppComponent)
	return item
}

func (c *AppCompClient) UpdateOrAdd(target *v1.AppComponent) *v1.AppComponent {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *AppCompClient) FetchList(s labels.Selector) []*v1.AppComponent {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("AppComponents").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("AppComponents").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*v1.AppComponent{}
	kList := rObj.(*v1.AppComponentList)
	for _, kItem := range kList.Items {
		target = append(target, &kItem)
	}

	return target
}

func (c *AppCompClient) TryDelete(target *v1.AppComponent) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("AppComponents").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("AppComponents").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *AppCompClient) Delete(target *v1.AppComponent) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *AppCompClient) List() []*v1.AppComponent {
	kItems := c.informer.GetStore().List()
	target := []*v1.AppComponent{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*v1.AppComponent))
	}
	return target
}
