package client

import (
	"k8s.io/kubernetes/pkg/controller/framework"
	"reflect"
)

// template type ClientTmpl(ResourceType)

type SitepodClient struct {
	supportedType reflect.Type
	informer      framework
}

func NewSitepodClient() *SitepodClient {
	return &SitepodClient{
		supportedType: reflect.TypeOf(&v1.Sitepod{}),
	}
}

func (c *SitepodClient) NewEmpty() *v1.Sitepod {
	return &v1.Sitepod{}
}

func (c *SitepodClient) GetByKey(key string) (*v1.Sitepod, bool, error) {
	iObj, exists, err := c.informer.GetStore().GetByKey(key)
	if iObj != nil {
		return nil, exists, err
	} else {
		return iObj.(*v1.Sitepod), exists, err
	}
}
