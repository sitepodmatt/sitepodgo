package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	"strings"
)

type HashedPassword struct {
	Scheme       string `json:"scheme,omitempty"`
	Salt         string `json:"salt,omitempty"`
	CombinedHash string `json:"combinedHash,omitempty"`
	Rounds       int    `json:"rounds,omitempty"`
}

func (hp HashedPassword) IsValid() bool {
	return len(hp.CombinedHash) > 0
}

type SystemUserSpec struct {
	Username string         `json:"username,omitempty"`
	Shell    string         `json:"shell,omitempty"`
	Password HashedPassword `json:"hashedPassword,omitempty"`
	Sitepod  string         `json:"sitepod,omitempty"`
}

type SystemUserStatus struct {
	AssignedFileUID int  `json:"assignedFileUID,omitempty"`
	HomeDirCreated  bool `json:"homeDirCreated,omitempty"`
}

type SystemUser struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 SystemUserSpec   `json:"spec"`
	Status               SystemUserStatus `json:"status"`
}

func (s *SystemUser) GetUsername() string {
	systemUsername := strings.TrimPrefix(s.Name, "systemuser-")
	//if systemUsername == s.Spec.Username {
	//panic("bad state. system username are expected to start with systemuser-")
	//}
	return systemUsername
}

func (s *SystemUser) GetHomeDirectory() string {
	//TODO this shouldnt be hardcoded and should be flexible strategy
	return "/home/" + s.GetUsername()
}

func (s *SystemUser) GetShell() string {
	//TODO this shouldnt be hardcoded and should be flexible strategy
	if len(s.Spec.Shell) > 0 {
		return s.Spec.Shell
	} else {
		return "/bin/bash"
	}
}

func (s *SystemUser) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

type SystemUserList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []SystemUser `json:"items"`
}

func (s *SystemUserList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}
