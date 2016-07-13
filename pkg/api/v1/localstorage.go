package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type LocalStorage struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 LocalStorageSpec   `json:"spec"`
	Status               LocalStorageStatus `json:"status"`
}

type LocalStorageSpec struct {
	NodeName string `json:"nodeName,omitempty"`
}

type LocalStorageStatus struct {
	RootPath string `json:"rootPath,omitempty"`
}

func (s *LocalStorage) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

type LocalStorageList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []LocalStorage `json:"items"`
}

func (s *LocalStorageList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}
