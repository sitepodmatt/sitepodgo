package v1

import (
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	"sitepod.io/sitepod/pkg/util"
)

type SitepodUserSpec struct {
	Email          string
	SaltedPassword string
	Salt           string
}

type SitepodUserStatus struct {
	Status string
}

type SitepodUser struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 SitepodUserSpec
	Status               SitepodUserStatus
}

type SitepodUserList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []SitepodUser `json:"items"`
}

func (s *SitepodUser) GetObjectMeta() meta.Object {
	om := v1.ObjectMeta(s.ObjectMeta)
	return &om
}

func (s *SitepodUser) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *SitepodUserList) GetListMeta() unversioned.List {
	lm := unversioned.ListMeta(s.ListMeta)
	return &lm
}

func (s *SitepodUser) SetDefaults() {
	s.ObjectMeta.Labels = make(map[string]string)
	s.ObjectMeta.Annotations = make(map[string]string)
	s.ObjectMeta.Namespace = "default"
}

func (s *SitepodUser) BeforeAdd() {

	if ptPasswd := s.ObjectMeta.Annotations["sitepod.io/plain-text-password"]; ptPasswd != "" {

		salt := util.RandomSalt(8)
		saltedPasswd, err := util.Sha512Crypt(ptPasswd, salt)

		if err != nil {
			panic(err)
		}

		s.Spec.Salt = salt
		s.Spec.SaltedPassword = saltedPasswd

		delete(s.ObjectMeta.Annotations, "sitepod.io/plain-text-password")

	}

}
