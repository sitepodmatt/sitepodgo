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
	resyncPeriodPVClaimClient = 5 * time.Minute
)

func HackImportIgnoredPVClaimClient(a k8s_api.Volume, b v1.Cluster, c1 ext_api.ThirdPartyResource) {
}

// template type ClientTmpl(ResourceType, ResourceName, ResourcePluralName, Namespaced, DefaultGenName)

type PVClaimClient struct {
	rc            *restclient.RESTClient
	ns            string
	supportedType reflect.Type
	informer      framework.SharedIndexInformer
}

func NewPVClaimClient(rc *restclient.RESTClient, ns string) *PVClaimClient {
	c := &PVClaimClient{
		rc:            rc,
		supportedType: reflect.TypeOf(&k8s_api.PersistentVolumeClaim{}),
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
		api.NewListWatchFromClient(c.rc, "PersistentVolumeClaims", c.ns, nil, pc),
		&k8s_api.PersistentVolumeClaim{},
		resyncPeriodPVClaimClient,
		indexers,
	)

	return c
}

func (c *PVClaimClient) StartInformer(stopCh <-chan struct{}) {
	c.informer.Run(stopCh)
}

func (c *PVClaimClient) AddInformerHandlers(reh framework.ResourceEventHandler) {
	if c.informer == nil {
		panic(fmt.Sprintf("%s informer not started", "PersistentVolumeClaim"))
	}

	c.informer.AddEventHandler(reh)
}

func (c *PVClaimClient) HasSynced() bool {
	if c.informer == nil {
		return false
	}
	return c.informer.HasSynced()
}

func (c *PVClaimClient) NewEmpty() *k8s_api.PersistentVolumeClaim {
	item := &k8s_api.PersistentVolumeClaim{}
	item.GenerateName = "sitepod-pvc-"
	return item
}

//TODO: wrong location? shared?
func (c *PVClaimClient) KeyOf(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

//TODO: wrong location? shared?
func (c *PVClaimClient) DeepEqual(a interface{}, b interface{}) bool {
	return k8s_api.Semantic.DeepEqual(a, b)
}

func (c *PVClaimClient) MaybeGetByKey(key string) (*k8s_api.PersistentVolumeClaim, bool) {

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
		item := iObj.(*k8s_api.PersistentVolumeClaim)
		glog.Infof("Got %s from informer store with rv %s", "PersistentVolumeClaim", item.ResourceVersion)
		return item, exists
	}
}

func (c *PVClaimClient) GetByKey(key string) *k8s_api.PersistentVolumeClaim {
	item, exists := c.MaybeGetByKey(key)

	if !exists {
		panic("Not found")
	}

	return item
}

func (c *PVClaimClient) BySitepodKey(sitepodKey string) ([]*k8s_api.PersistentVolumeClaim, error) {
	items, err := c.informer.GetIndexer().ByIndex("sitepod", sitepodKey)

	if err != nil {
		return nil, err
	}

	typedItems := []*k8s_api.PersistentVolumeClaim{}
	for _, item := range items {
		typedItems = append(typedItems, item.(*k8s_api.PersistentVolumeClaim))
	}
	return typedItems, nil
}

func (c *PVClaimClient) SingleBySitepodKey(sitepodKey string) *k8s_api.PersistentVolumeClaim {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		panic(errors.New("None found"))
	}

	return items[0]

}

func (c *PVClaimClient) MaybeSingleBySitepodKey(sitepodKey string) (*k8s_api.PersistentVolumeClaim, bool) {

	items, err := c.BySitepodKey(sitepodKey)

	if err != nil {
		panic(err)
	}

	if len(items) == 0 {
		return nil, false
	} else {

		if len(items) > 1 {
			glog.Warningf("Unexpected number of %s for sitepod %s - %d items matched", "PersistentVolumeClaims", sitepodKey, len(items))
		}

		return items[0], true
	}

}

func (c *PVClaimClient) Add(target *k8s_api.PersistentVolumeClaim) *k8s_api.PersistentVolumeClaim {

	rcReq := c.rc.Post()
	if true {
		rcReq = rcReq.Namespace(c.ns)
	}

	result := rcReq.Resource("PersistentVolumeClaims").Body(target).Do()

	if err := result.Error(); err != nil {
		panic(err)
	}

	r, err := result.Get()

	if err != nil {
		panic(err)
	}
	item := r.(*k8s_api.PersistentVolumeClaim)
	glog.Infof("Added %s - %s (rv: %s)", "PersistentVolumeClaim", item.Name, item.ResourceVersion)
	return item
}

func (c *PVClaimClient) UpdateOrAdd(target *k8s_api.PersistentVolumeClaim) *k8s_api.PersistentVolumeClaim {

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
		replacementTarget, err := rcReq.Resource("PersistentVolumeClaims").Name(rName).Body(target).Do().Get()
		if err != nil {
			panic(err)
		}
		item := replacementTarget.(*k8s_api.PersistentVolumeClaim)
		glog.Infof("Updated %s - %s (rv: %s)", "PersistentVolumeClaim", item.Name, item.ResourceVersion)
		return item
	} else {
		return c.Add(target)
	}
}
