package v1

import (
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
)

type AppComponent struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 AppComponentSpec   `json:"spec"`
	Status               AppComponentStatus `json:"status"`
}

type AppComponentConfigFile struct {
	Path     string `json:"path,omitempty"`
	Template string `json:"content,omitempty"`
	FileMode string `json:"fileMode,omitempty"`
	Uid      int    `json:"uid,omitempty"`
	Gid      int    `json:"gid,omitempty"`
}

type AppComponentSpec struct {
	Type           string                   `json:"type,omitempty"`
	DisplayName    string                   `json:"displayName,omitempty"`
	Image          string                   `json:"image,omitempty"`
	ImageVersion   string                   `json:"imageVersion,omitempty"`
	Description    string                   `json:"description,omitempty"`
	DataVolumeName string                   `json:"dataVolumeName,omitempty"`
	EtcMergeMode   string                   `json:"etcMergeMode,omitempty"`
	ConfigFiles    []AppComponentConfigFile `json:"configFiles,omitempty"`
	//TODO move these out
	PrivateKeyPEM string `json:"privateKeyPEM,omitempty"`
	PublicKeyPEM  string `json:"publicKeyPEM,omitempty"`
}

type AppComponentStatus struct {
	//TODO figure out high level conditions
}

func (s *AppComponent) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *AppComponent) GetObjectMeta() meta.Object {
	om := v1.ObjectMeta(s.ObjectMeta)
	return &om
}

type AppComponentList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []AppComponent `json:"items"`
}

func (s *AppComponentList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *AppComponentList) GetListMeta() unversioned.List {
	lm := unversioned.ListMeta(s.ListMeta)
	return &lm
}
