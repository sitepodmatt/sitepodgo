package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type ListOptions struct {
	unversioned.TypeMeta `json:",inline"`

	// A selector to restrict the list of returned objects by their labels.
	// Defaults to everything.
	LabelSelector string `json:"labelSelector,omitempty" protobuf:"bytes,1,opt,name=labelSelector"`
	// A selector to restrict the list of returned objects by their fields.
	// Defaults to everything.
	FieldSelector string `json:"fieldSelector,omitempty" protobuf:"bytes,2,opt,name=fieldSelector"`
	// Watch for changes to the described resources and return them as a stream of
	// add, update, and remove notifications. Specify resourceVersion.
	Watch bool `json:"watch,omitempty" protobuf:"varint,3,opt,name=watch"`
	// When specified with a watch call, shows changes that occur after that particular version of a resource.
	// Defaults to changes from the beginning of history.
	ResourceVersion string `json:"resourceVersion,omitempty" protobuf:"bytes,4,opt,name=resourceVersion"`
	// Timeout for the list/watch call.
	TimeoutSeconds *int64 `json:"timeoutSeconds,omitempty" protobuf:"varint,5,opt,name=timeoutSeconds"`
}

func (lo *ListOptions) GetObjectKind() unversioned.ObjectKind {
	return &lo.TypeMeta
}
