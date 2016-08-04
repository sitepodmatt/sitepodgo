package v1

import (
	"errors"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
)

type Sitepod struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 SitepodSpec   `json:"spec"`
	Status               SitepodStatus `json:"status"`
}

type SitepodSpec struct {
	DisplayName  string   `json:"displayName,omitempty"`
	Description  string   `json:"description,omitempty"`
	VolumeClaims []string `json:"volumeClaims,omitempty"`
}

type SitepodStatus struct {
	Pods         []string `json:"pods,omitempty"`
	LocalStorage []string `json:"localStorage,omitempty"`
}

func (s *Sitepod) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *Sitepod) GetObjectMeta() meta.Object {
	om := v1.ObjectMeta(s.ObjectMeta)
	return &om
}

func (s *Sitepod) GetRootStorageName() (string, error) {

	if len(s.Status.LocalStorage) == 0 {
		return "", errors.New("No local storage provisioned")
	}

	return s.Status.LocalStorage[0], nil
}

type SitepodList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []Sitepod `json:"items"`
}

func (s *SitepodList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *SitepodList) GetListMeta() unversioned.List {
	lm := unversioned.ListMeta(s.ListMeta)
	return &lm
}

//func (s *SitepodList) GetListMeta() .List {
//return &s.ListMeta
//}
