package v1

import (
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
)

//TODO third party resource dont seem to support capitalization of say appComponent
// so for now its Appcomponent

type Appcomponent struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 AppComponentSpec   `json:"spec"`
	Status               AppComponentStatus `json:"status"`
}

type AppComponentConfigFile struct {
	Name      string `json:"name"`
	Directory string `json:"directory,omitempty"`
	Filename  string `json:"filename,omitempty"`
	Content   string `json:"content,omitempty"`
	FileMode  string `json:"fileMode,omitempty"`
	Uid       int    `json:"uid,omitempty"`
	Gid       int    `json:"gid,omitempty"`
}

type AppComponentSpec struct {
	Type             string `json:"type,omitempty"`
	DisplayName      string `json:"displayName,omitempty"`
	Description      string `json:"description,omitempty"`
	Image            string `json:"image,omitempty"`
	ImageVersion     string `json:"imageVersion,omitempty"`
	Expose           bool   `json:"expose,omitempty"`
	ExposePort       int32  `json:"exposePort,omitempty"`
	ExposeExternally bool
	MountHome        bool                     `json:"mountHome,omitempty"`
	MountEtcs        bool                     `json:"mountEtcs,omitempty"`
	ConfigFiles      []AppComponentConfigFile `json:"configFiles,omitempty"`
}

type AppComponentStatus struct {
	//TODO figure out high level conditions
}

func (s *Appcomponent) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *Appcomponent) GetObjectMeta() meta.Object {
	om := v1.ObjectMeta(s.ObjectMeta)
	return &om
}

type AppcomponentList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []Appcomponent `json:"items"`
}

func (s *AppcomponentList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *AppcomponentList) GetListMeta() unversioned.List {
	lm := unversioned.ListMeta(s.ListMeta)
	return &lm
}
