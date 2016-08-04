package client

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	k8s_api "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
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
	resyncPeriodSitepodClient = 5 * time.Minute
)

func HackImportIgnoredSitepodClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type SitepodClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewSitepodClient(rc *restclient.RESTClient, ns string) *SitepodClient {
	c := &SitepodClient{
		rc:            rc,
		supportedType: reflect.TypeOf(&v1.Sitepod{}),
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
		api.NewListWatchFromClient(c.rc, "Sitepods", c.ns, nil, pc),
		&v1.Sitepod{},
		resyncPeriodSitepodClient,
		indexers,
	)

	return c
}

func (c *SitepodClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *SitepodClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "Sitepod"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *SitepodClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *SitepodClient) NewEmpty() *v1.Sitepod {
	item := &v1.Sitepod{}
	item.GenerateName = "sitepod-"
	return item
}

//TODO: wrong location? shared?
func (c *SitepodClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

//TODO: wrong location? shared?
func (c *SitepodClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *SitepodClient) MaybeGetByKey(key string) (*v1.Sitepod, bool) {

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
		item := iObj.(*v1.Sitepod)
		glog.Infof("Got %s from informer store with rv %s", "Sitepod", item.ResourceVersion)
		return item, exists
	}
}

func (c *SitepodClient) GetByKey(key string) *v1.Sitepod {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *SitepodClient) BySitepodKey(sitepodKey string) ([]*v1.Sitepod, error) {
	items, err := c.informer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		return nil, err
	}

	typedItems := []*v1.Sitepod{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*v1.Sitepod))
	}
	return typedItems, nil
}

func (c *SitepodClient) SingleBySitepodKey(sitepodKey string) *v1.Sitepod {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *SitepodClient) MaybeSingleBySitepodKey(sitepodKey string) (*v1.Sitepod, bool) {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "Sitepods", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *SitepodClient) Add(target *v1.Sitepod) *v1.Sitepod {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("Sitepods").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*v1.Sitepod)
	glog.Infof("Added %s - %s (rv: %s)", "Sitepod", item.Name, item.ResourceVersion)
	return item
}

func (c *SitepodClient) UpdateOrAdd(target *v1.Sitepod) *v1.Sitepod {

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
		replacementTarget, err := rcReq.Resource("Sitepods").Name(rName).Body(target).Do().Get()
		if err != nil {
			glog.Errorf("Type of error: %+v : %s", err, reflect.TypeOf(err))
			errResult := err.(*kerrors.StatusError)
			glog.Errorf("Status code: %s ", errResult.ErrStatus.Reason)
			panic(err)
		}
		item := replacementTarget.(*v1.Sitepod)
		glog.Infof("Updated %s - %s (rv: %s)", "Sitepod", item.Name, item.ResourceVersion)
		return item
	} else {
		return c.Add(target)
	}
}
