package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type Cluster struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 ClusterSpec   `json:"spec"`
	Status               ClusterStatus `json:"status"`
}

type ClusterSpec struct {
	DisplayName  string `json:"displayName,omitempty"`
	Description  string `json:"description,omitempty"`
	FileUIDCount int    `json:"fileUidCount"`
}

func (s *Cluster) NextFileUID() int {
	//NOTE we need atomic increment here if more than one worker
	s.Spec.FileUIDCount = s.Spec.FileUIDCount + 1
	return s.Spec.FileUIDCount
}

type ClusterStatus struct{}

func (s *Cluster) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

type ClusterList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []Cluster `json:"items"`
}

func (s *ClusterList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}
