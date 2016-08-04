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
	resyncPeriodPVClient = 5 * time.Minute
)

func HackImportIgnoredPVClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type PVClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewPVClient(rc *restclient.RESTClient, ns string) *PVClient {
	c := &PVClient{
		rc:            rc,
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

func (c *PVClient) NewEmpty() *k8s_api.PersistentVolume {
	item := &k8s_api.PersistentVolume{}
	item.GenerateName = "sitepod-pv-"
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
		item := iObj.(*k8s_api.PersistentVolume)
		glog.Infof("Got %s from informer store with rv %s", "PersistentVolume", item.ResourceVersion)
		return item, exists
	}
}

func (c *PVClient) GetByKey(key string) *k8s_api.PersistentVolume {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *PVClient) BySitepodKey(sitepodKey string) ([]*k8s_api.PersistentVolume, error) {
	items, err := c.informer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		return nil, err
	}

	typedItems := []*k8s_api.PersistentVolume{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*k8s_api.PersistentVolume))
	}
	return typedItems, nil
}

func (c *PVClient) SingleBySitepodKey(sitepodKey string) *k8s_api.PersistentVolume {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *PVClient) MaybeSingleBySitepodKey(sitepodKey string) (*k8s_api.PersistentVolume, bool) {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "PersistentVolumes", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *PVClient) Add(target *k8s_api.PersistentVolume) *k8s_api.PersistentVolume {

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

func (c *PVClient) UpdateOrAdd(target *k8s_api.PersistentVolume) *k8s_api.PersistentVolume {

	accessor, err := meta.Accessor(target)
	if err != nil {
		panic(err)
	}

	uid := accessor.GetUID()
	if len(string(uid)) > 0 {
		rName := accessor.GetName()
		rcReq := c.rc.Put()
		if false {
			rcReq = rcReq.Namespace(c.ns)
		}
		replacementTarget, err := rcReq.Resource("PersistentVolumes").Name(rName).Body(target).Do().Get()
		if err != nil {
			glog.Errorf("Type of error: %+v : %s", err, reflect.TypeOf(err))
			errResult := err.(*kerrors.StatusError)
			glog.Errorf("Status code: %s ", errResult.ErrStatus.Reason)
			panic(err)
		}
		item := replacementTarget.(*k8s_api.PersistentVolume)
		glog.Infof("Updated %s - %s (rv: %s)", "PersistentVolume", item.Name, item.ResourceVersion)
		return item
	} else {
		return c.Add(target)
	}
}
