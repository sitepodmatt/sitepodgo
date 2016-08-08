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
	resyncPeriodDeploymentClient = 5 * time.Minute
)

func HackImportIgnoredDeploymentClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceListType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type ResouceListTypeDeploymentClient []int

type DeploymentClient struct {
	rc            *restclient.RESTClient
	rcConfig      *restclient.Config
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewDeploymentClient(rc *restclient.RESTClient, config *restclient.Config, ns string) *DeploymentClient {
	c := &DeploymentClient{
		rc:            rc,
		rcConfig:      config,
		supportedType: reflect.TypeOf(&ext_api.Deployment{}),
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
		api.NewListWatchFromClient(c.rc, "Deployments", c.ns, nil, pc),
		&ext_api.Deployment{},
		resyncPeriodDeploymentClient,
		indexers,
	)

	return c
}

func (c *DeploymentClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *DeploymentClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "Deployment"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *DeploymentClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *DeploymentClient) NewEmpty() *ext_api.Deployment {
	item := &ext_api.Deployment{}
	item.GenerateName = "sitepod-deployment-"
	return item
}

//TODO: wrong location? shared?
func (c *DeploymentClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

//TODO: wrong location? shared?
func (c *DeploymentClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *DeploymentClient) MaybeGetByKey(key string) (*ext_api.Deployment, bool) {

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
		glog.Infof("Got %s from informer store with rv %s", "Deployment", item.ResourceVersion)
		return item, exists
	}
}

func (c *DeploymentClient) GetByKey(key string) *ext_api.Deployment {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *DeploymentClient) ByIndexByKey(index string, key string) []*ext_api.Deployment {

	items, err := c.informer.GetIndexer().ByIndex(index, key)

	if err != nil {
		panic(err)
	}

	typedItems := []*ext_api.Deployment{}
	for _, item := range items {
		typedItems = append(typedItems, c.CloneItem(item))
	}
	return typedItems
}

func (c *DeploymentClient) BySitepodKey(sitepodKey string) []*ext_api.Deployment {
	return c.ByIndexByKey("sitepod", sitepodKey)
}

func (c *DeploymentClient) MaybeSingleByUID(uid string) (*ext_api.Deployment, bool) {
	items := c.ByIndexByKey("uid", uid)
	if len(items) == 0 {
		return nil, false
	} else {
		return items[0], true
	}
}

func (c *DeploymentClient) SingleBySitepodKey(sitepodKey string) *ext_api.Deployment {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *DeploymentClient) MaybeSingleBySitepodKey(sitepodKey string) (*ext_api.Deployment, bool) {

	items := c.BySitepodKey(sitepodKey)

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "Deployments", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *DeploymentClient) Add(target *ext_api.Deployment) *ext_api.Deployment {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("Deployments").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*ext_api.Deployment)
	glog.Infof("Added %s - %s (rv: %s)", "Deployment", item.Name, item.ResourceVersion)
	return item
}

func (c *DeploymentClient) CloneItem(orig interface{}) *ext_api.Deployment {
	cloned, err := conversion.NewCloner().DeepCopy(orig)
	if err != nil {
		panic(err)
	}
	return cloned.(*ext_api.Deployment)
}

func (c *DeploymentClient) Update(target *ext_api.Deployment) *ext_api.Deployment {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}
	rName := accessor.GetName()
	rcReq := c.rc.Put()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}
	replacementTarget, err := rcReq.Resource("Deployments").Name(rName).Body(target).Do().Get()
	if err != nil {
		panic(err)
	}
	item := replacementTarget.(*ext_api.Deployment)
	return item
}

func (c *DeploymentClient) UpdateOrAdd(target *ext_api.Deployment) *ext_api.Deployment {

	if len(string(target.UID)) > 0 {
		return c.Update(target)
	} else {
		return c.Add(target)
	}
}

func (c *DeploymentClient) FetchList(s labels.Selector) []*ext_api.Deployment {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Get().Resource("Deployments").LabelsSelectorParam(s)
	} else {
		prc = c.rc.Get().Resource("Deployments").Namespace(c.ns).LabelsSelectorParam(s)
	}

	rObj, err := prc.Do().Get()

	if err != nil {
		panic(err)
	}

	target := []*ext_api.Deployment{}
	kList := rObj.(*ext_api.DeploymentList)
	for _, kItem := range kList.Items {
		target = append(target, c.CloneItem(&kItem))
	}

	return target
}

func (c *DeploymentClient) TryDelete(target *ext_api.Deployment) error {

	var prc *restclient.Request
	if !true {
		prc = c.rc.Delete().Resource("Deployments").Name(target.Name)
	} else {
		prc = c.rc.Delete().Namespace(c.ns).Resource("Deployments").Name(target.Name)
	}

	err := prc.Do().Error()
	return err
}

func (c *DeploymentClient) Delete(target *ext_api.Deployment) {

	err := c.TryDelete(target)

	if err != nil {
		panic(err)
	}
}

func (c *DeploymentClient) List() []*ext_api.Deployment {
	kItems := c.informer.GetStore().List()
	target := []*ext_api.Deployment{}
	for _, kItem := range kItems {
		target = append(target, kItem.(*ext_api.Deployment))
	}
	return target
}

func (c *DeploymentClient) RestClient() *restclient.RESTClient {
	return c.rc
}

func (c *DeploymentClient) RestClientConfig() *restclient.Config {
	return c.rcConfig
}
