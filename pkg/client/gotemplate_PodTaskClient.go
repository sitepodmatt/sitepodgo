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
	resyncPeriodPodTaskClient = 5 * time.Minute
)

func HackImportIgnoredPodTaskClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypePodTaskClient []int

type PodTaskClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewPodTaskClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *PodTaskClient {
	c := &PodTaskClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&v1.Podtask{}),
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
		api.NewListWatchFromClient(c.rc, "PodTasks", c.ns, nil, pc),
		&v1.Podtask{},
		resyncPeriodPodTaskClient,
		indexers,
	)

	return c
}

func (c *PodTaskClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *PodTaskClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "PodTask"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *PodTaskClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *PodTaskClient) NewEmpty() *v1.Podtask {
	item := &v1.Podtask{}
	item.GenerateName = "sitepod-podtask-"
	return item
}

//TODO: wrong location? shared?
func (c *PodTaskClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

//TODO: wrong location? shared?
func (c *PodTaskClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *PodTaskClient) MaybeGetByKey(key string) (*v1.Podtask, bool) {

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
		glog.Infof("Got %s from informer store with rv %s", "PodTask", item.ResourceVersion)
		return item, exists
	}
}

func (c *PodTaskClient) GetByKey(key string) *v1.Podtask {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *PodTaskClient) ByIndexByKey(index string, key string) []*v1.Podtask {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*v1.Podtask{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *PodTaskClient) BySitepodKey(sitepodKey string) []*v1.Podtask {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *PodTaskClient) MaybeSingleByUID(uid string) (*v1.Podtask, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *PodTaskClient) SingleBySitepodKey(sitepodKey string) *v1.Podtask {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *PodTaskClient) MaybeSingleBySitepodKey(sitepodKey string) (*v1.Podtask, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "PodTasks", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *PodTaskClient) Add(target *v1.Podtask) *v1.Podtask {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("PodTasks").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*v1.Podtask)
	glog.Infof("Added %s - %s (rv: %s)", "PodTask", item.Name, item.ResourceVersion)
	return item
}

func (c *PodTaskClient) CloneItem(orig interface{}) *v1.Podtask {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*v1.Podtask)
}

func (c *PodTaskClient) Update(target *v1.Podtask) *v1.Podtask {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("PodTasks").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*v1.Podtask)
	return item
}

func (c *PodTaskClient) UpdateOrAdd(target *v1.Podtask) *v1.Podtask {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *PodTaskClient) FetchList(s labels.Selector) []*v1.Podtask {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("PodTasks").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("PodTasks").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*v1.Podtask{}
	kList := rObj.(*v1.PodtaskList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *PodTaskClient) TryDelete(target *v1.Podtask) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("PodTasks").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("PodTasks").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *PodTaskClient) Delete(target *v1.Podtask) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *PodTaskClient) List() []*v1.Podtask {
	kItems := c.informer.GetStore().List()
	target := []*v1.Podtask{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*v1.Podtask))
	}
	return target
}

func (c *PodTaskClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *PodTaskClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
