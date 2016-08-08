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
	resyncPeriodClusterClient = 5 * time.Minute
)

func HackImportIgnoredClusterClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeClusterClient []int

type ClusterClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewClusterClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *ClusterClient {
	c := &ClusterClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&v1.Cluster{}),
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
		api.NewListWatchFromClient(c.rc, "Clusters", c.ns, nil, pc),
		&v1.Cluster{},
		resyncPeriodClusterClient,
		indexers,
	)

	return c
}

func (c *ClusterClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *ClusterClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "Cluster"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *ClusterClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *ClusterClient) NewEmpty() *v1.Cluster {
	item := &v1.Cluster{}
	item.GenerateName = "sitepod-cluster-"
	return item
}

//TODO: wrong location? shared?
func (c *ClusterClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

//TODO: wrong location? shared?
func (c *ClusterClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *ClusterClient) MaybeGetByKey(key string) (*v1.Cluster, bool) {

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
		glog.Infof("Got %s from informer store with rv %s", "Cluster", item.ResourceVersion)
		return item, exists
	}
}

func (c *ClusterClient) GetByKey(key string) *v1.Cluster {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *ClusterClient) ByIndexByKey(index string, key string) []*v1.Cluster {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*v1.Cluster{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *ClusterClient) BySitepodKey(sitepodKey string) []*v1.Cluster {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *ClusterClient) MaybeSingleByUID(uid string) (*v1.Cluster, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *ClusterClient) SingleBySitepodKey(sitepodKey string) *v1.Cluster {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *ClusterClient) MaybeSingleBySitepodKey(sitepodKey string) (*v1.Cluster, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "Clusters", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *ClusterClient) Add(target *v1.Cluster) *v1.Cluster {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("Clusters").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*v1.Cluster)
	glog.Infof("Added %s - %s (rv: %s)", "Cluster", item.Name, item.ResourceVersion)
	return item
}

func (c *ClusterClient) CloneItem(orig interface{}) *v1.Cluster {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*v1.Cluster)
}

func (c *ClusterClient) Update(target *v1.Cluster) *v1.Cluster {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("Clusters").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*v1.Cluster)
	return item
}

func (c *ClusterClient) UpdateOrAdd(target *v1.Cluster) *v1.Cluster {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *ClusterClient) FetchList(s labels.Selector) []*v1.Cluster {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("Clusters").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("Clusters").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*v1.Cluster{}
	kList := rObj.(*v1.ClusterList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *ClusterClient) TryDelete(target *v1.Cluster) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("Clusters").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("Clusters").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *ClusterClient) Delete(target *v1.Cluster) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *ClusterClient) List() []*v1.Cluster {
	kItems := c.informer.GetStore().List()
	target := []*v1.Cluster{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*v1.Cluster))
	}
	return target
}

func (c *ClusterClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *ClusterClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
