package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type CounterSpec struct {
	Count int `json:"count,omitempty"`
}

type Counter struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 CounterSpec `json:"spec"`
}

func (s *Counter) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *Counter) NextCount() int {
	//TODO do we need atomic increment here?
	s.Spec.Count = s.Spec.Count + 1
	return s.Spec.Count
}
