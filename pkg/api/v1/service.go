package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type Serviceinstance struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 ServiceinstanceSpec   `json:"spec"`
	Status               ServiceinstanceStatus `json:"status"`
}

type ServiceConfigFile struct {
	Path     string `json:"path,omitempty"`
	Template string `json:"content,omitempty"`
	FileMode string `json:"fileMode,omitempty"`
	Uid      int    `json:"uid,omitempty"`
	Gid      int    `json:"gid,omitempty"`
}

type ServiceinstanceSpec struct {
	Type           string              `json:"type,omitempty"`
	DisplayName    string              `json:"displayName,omitempty"`
	Image          string              `json:"image,omitempty"`
	ImageVersion   string              `json:"imageVersion,omitempty"`
	Description    string              `json:"description,omitempty"`
	DataVolumeName string              `json:"dataVolumeName,omitempty"`
	EtcMergeMode   string              `json:"etcMergeMode,omitempty"`
	ConfigFiles    []ServiceConfigFile `json:"configFiles,omitempty"`
	PrivateKeyPEM  string              `json:"privateKeyPEM,omitempty"`
	PublicKeyPEM   string              `json:"publicKeyPEM,omitempty"`
}

type ServiceinstanceStatus struct {
}

func (s *Serviceinstance) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

type ServiceinstanceList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []Serviceinstance `json:"items"`
}

func (s *ServiceinstanceList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}
