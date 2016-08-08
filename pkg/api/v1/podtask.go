package v1

import (
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
)

type Podtask struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 PodTaskSpec   `json:"spec"`
	Status               PodTaskStatus `json:"status"`
}

type PodTaskSpec struct {
	Namespace     string   `json:"namespace"`
	PodName       string   `json:"podName"`
	ContainerName string   `json:"containerName",omitempty`
	Command       []string `json:"command"`
	MaxAttempts   int      `json:"mergeErrAndOut,omitempty"`
}

type PodTaskStatus struct {
	Completed bool   `json:"completed"`
	Attempts  int    `json:"attempt"`
	ExitCode  int    `json:"exitCode"`
	StdErr    string `json:"stdErr"`
	StdOut    string `json:"stdOut"`
}

func (s *Podtask) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *Podtask) GetObjectMeta() meta.Object {
	om := v1.ObjectMeta(s.ObjectMeta)
	return &om
}

type PodtaskList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []Podtask `json:"items"`
}

func (s *PodtaskList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *PodtaskList) GetListMeta() unversioned.List {
	lm := unversioned.ListMeta(s.ListMeta)
	return &lm
}
