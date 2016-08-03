package clienttmpl

import (
	"k8s.io/kubernetes/pkg/controller/framework"
	"reflect"
)

// template type ClientTmpl(ResourceType)
type ResourceType int

type ClientTmpl struct {
	supportedType reflect.Type
	informer      framework
}

func NewClientTmpl() *ClientTmpl {
	return &ClientTmpl{
		supportedType: reflect.TypeOf(&ResourceType{}),
	}
}

func (c *ClientTmpl) NewEmpty() *ResourceType {
	return &ResourceType{}
}

func (c *ClientTmpl) MaybeGetByKey(key string) (*ResourceType, bool, error) {
	iObj, exists, err := c.informer.GetStore().GetByKey(key)
	if iObj != nil {
		return nil, exists, err
	} else {
		return iObj.(*ResourceType), exists, err
	}
})

func (c *ClientTmpl) GetByKey(key string) *ResourceType {
	item, exists, err = c.MaybeGetByKey(key)

	if err != nil {
		panic(err)
	}

	if !exists {
		panic("Not found")
	}

	return item
}

func BySitepodKey(sitepodKey string) ([]*ResourceType, error) {
}

func SingleBySitepodKey(sitepodKey string) (*ResourceType, error) {

}

func UpdateOr(target *ResourceType, func() or,) *ResourceType {
	//panic
}


