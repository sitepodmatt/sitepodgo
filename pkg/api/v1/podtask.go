package v1

import (
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
)

type PodTask struct {
	unversioned.TypeMeta `json:",inline"`
	ObjectMeta           `json:"metadata,omitempty"`
	Spec                 PodTaskSpec   `json:"spec"`
	Status               PodTaskStatus `json:"status"`
}

type PodTaskSpec struct {
	Namespace      string   `json:"namespace"`
	PodName        string   `json:"podName"`
	ContainerName  string   `json:"containerName",omitempty`
	Command        []string `json:"command"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"`
	MergeErrAndOut bool     `json:"mergeErrAndOut,omitempty"`
}

type PodTaskStatus struct {
	ExitCode int    `json:"exitCode"`
	StdErr   string `json:"stdErr"`
	StdOut   string `json:"stdOut"`
}

func (s *PodTask) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *PodTask) GetObjectMeta() meta.Object {
	om := v1.ObjectMeta(s.ObjectMeta)
	return &om
}

type PodTaskList struct {
	unversioned.TypeMeta `json:",inline"`
	ListMeta             `json:"metadata,omitempty"`
	Items                []Cluster `json:"items"`
}

func (s *PodTaskList) GetObjectKind() unversioned.ObjectKind {
	return &s.TypeMeta
}

func (s *PodTaskList) GetListMeta() unversioned.List {
	lm := unversioned.ListMeta(s.ListMeta)
	return &lm
}
