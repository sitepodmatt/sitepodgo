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

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type DeploymentClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewDeploymentClient(rc *restclient.RESTClient, ns string) *DeploymentClient {
	c := &DeploymentClient{
		rc:            rc,
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
		item := iObj.(*ext_api.Deployment)
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

func (c *DeploymentClient) BySitepodKey(sitepodKey string) ([]*ext_api.Deployment, error) {
	items, err := c.informer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		return nil, err
	}

	typedItems := []*ext_api.Deployment{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*ext_api.Deployment))
	}
	return typedItems, nil
}

func (c *DeploymentClient) SingleBySitepodKey(sitepodKey string) *ext_api.Deployment {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *DeploymentClient) MaybeSingleBySitepodKey(sitepodKey string) (*ext_api.Deployment, bool) {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

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

func (c *DeploymentClient) UpdateOrAdd(target *ext_api.Deployment) *ext_api.Deployment {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}

	uid := accessor.GetUID()
	if len(string(uid)) > 0 {
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
		glog.Infof("Updated %s - %s (rv: %s)", "Deployment", item.Name, item.ResourceVersion)
		return item
	} else {
		return c.Add(target)
	}
}
