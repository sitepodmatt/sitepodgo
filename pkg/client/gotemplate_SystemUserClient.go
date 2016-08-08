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
	resyncPeriodSystemUserClient = 5 * time.Minute
)

func HackImportIgnoredSystemUserClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeSystemUserClient []int

type SystemUserClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewSystemUserClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *SystemUserClient {
	c := &SystemUserClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&v1.SystemUser{}),
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
		api.NewListWatchFromClient(c.rc, "SystemUsers", c.ns, nil, pc),
		&v1.SystemUser{},
		resyncPeriodSystemUserClient,
		indexers,
	)

	return c
}

func (c *SystemUserClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *SystemUserClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "SystemUser"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *SystemUserClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *SystemUserClient) NewEmpty() *v1.SystemUser {
	item := &v1.SystemUser{}
	item.GenerateName = "systemuser-"
	return item
}

//TODO: wrong location? shared?
func (c *SystemUserClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

//TODO: wrong location? shared?
func (c *SystemUserClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *SystemUserClient) MaybeGetByKey(key string) (*v1.SystemUser, bool) {

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
		item := iObj.(*v1.SystemUser)
		glog.Infof("Got %s from informer store with rv %s", "SystemUser", item.ResourceVersion)
		return item, exists
	}
}

func (c *SystemUserClient) GetByKey(key string) *v1.SystemUser {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *SystemUserClient) ByIndexByKey(index string, key string) []*v1.SystemUser {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*v1.SystemUser{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*v1.SystemUser))
	}
	return typedItems
}

func (c *SystemUserClient) BySitepodKey(sitepodKey string) []*v1.SystemUser {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *SystemUserClient) MaybeSingleByUID(uid string) (*v1.SystemUser, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *SystemUserClient) SingleBySitepodKey(sitepodKey string) *v1.SystemUser {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *SystemUserClient) MaybeSingleBySitepodKey(sitepodKey string) (*v1.SystemUser, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "SystemUsers", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *SystemUserClient) Add(target *v1.SystemUser) *v1.SystemUser {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("SystemUsers").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*v1.SystemUser)
	glog.Infof("Added %s - %s (rv: %s)", "SystemUser", item.Name, item.ResourceVersion)
	return item
}

func (c *SystemUserClient) Update(target *v1.SystemUser) *v1.SystemUser {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("SystemUsers").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*v1.SystemUser)
	return item
}

func (c *SystemUserClient) UpdateOrAdd(target *v1.SystemUser) *v1.SystemUser {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *SystemUserClient) FetchList(s labels.Selector) []*v1.SystemUser {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("SystemUsers").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("SystemUsers").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*v1.SystemUser{}
	kList := rObj.(*v1.SystemUserList)
	for _, kItem := range kList.Items {
		target = append(target, &kItem)
	}

	return target
}

func (c *SystemUserClient) TryDelete(target *v1.SystemUser) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("SystemUsers").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("SystemUsers").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *SystemUserClient) Delete(target *v1.SystemUser) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *SystemUserClient) List() []*v1.SystemUser {
	kItems := c.informer.GetStore().List()
	target := []*v1.SystemUser{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*v1.SystemUser))
	}
	return target
}

func (c *SystemUserClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *SystemUserClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
