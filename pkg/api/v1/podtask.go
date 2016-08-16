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

const (
	DefaultMaxAttempts = 5
)

func (p *Podtask) SetDefaults() {
	p.ObjectMeta.Labels = make(map[string]string)
	p.ObjectMeta.Annotations = make(map[string]string)
	p.Spec.MaxAttempts = DefaultMaxAttempts
}

type PodTaskSpec struct {
	Namespace       string   `json:"namespace"`
	PodName         string   `json:"podName"`
	ContainerName   string   `json:"containerName"`
	Command         []string `json:"command"`
	MaxAttempts     int      `json:"maxAttempts"`
	BehalfType      string   `json:"behalfType,omitempty"`
	BehalfOf        string   `json:"behalfOf,omitempty"`
	BehalfCondition string   `json:"behalfCondition,omitempty"`
}

type PodTaskStatus struct {
	Completed bool   `json:"completed"`
	Attempts  int    `json:"attempts"`
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
