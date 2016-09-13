package v1

import (
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
)

type Website struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 WebsiteSpec   `json:"spec"`
	Status               WebsiteStatus `json:"status"`
}

func (c *Website) SetDefaults() {
	c.ObjectMeta.Labels = make(map[string]string)
	c.ObjectMeta.Annotations = make(map[string]string)
}

func (c *Website) GetPrimaryDomain() string {
	return ""
}

type WebsiteSpec struct {
	Name string `json:",inline"`
}

func (s *Website) GetObjectMeta() meta.Object {
	om := v1.ObjectMeta(s.ObjectMeta)
	return &om
}

type WebsiteStatus struct {
	DirectoryCreated  bool
	SkeltonSetup      bool
	ServerSetup       bool
	LoadBalancerSetup bool
}

func (s *Website) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

type WebsiteList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []Website `json:"items"`
}

func (s *WebsiteList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *WebsiteList) GetListMeta() unversioned.List {
	lm := unversioned.ListMeta(s.ListMeta)
	return &lm
}
