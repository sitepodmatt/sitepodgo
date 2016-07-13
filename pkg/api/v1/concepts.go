package v1

import (
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"sitepod.io/sitepod/pkg/api"
)

type UpdaterFunc func(runtime.Object) (runtime.Object, error)
type AdderFunc func(runtime.Object) (runtime.Object, error)
type GetterFunc func(string) (runtime.Object, error)
type DeleterFunc func(runtime.Object) error
type ListLabelFunc func(labels.Selector) (runtime.Object, error)

type ConceptFuncs struct {
	Get           func() *restclient.Request
	GetWithLabels func(labels.Selector) (runtime.Object, error)
	Post          func() *restclient.Request
	Put           func() *restclient.Request
	Delete        func() *restclient.Request
	Updater       func(runtime.Object) (runtime.Object, error)
	Adder         func(runtime.Object) (runtime.Object, error)
	Deleter       func(runtime.Object) error
	Getter        func(string) (runtime.Object, error)
	ListWatcher   func(fields.Selector) cache.ListerWatcher
	Type          func() runtime.Object
	KindSingle    func() string
}

type Concepts struct {
	SystemUsers *ConceptFuncs
	Counters    *ConceptFuncs
	Sitepods    *ConceptFuncs
	Services    *ConceptFuncs
	Clusters    *ConceptFuncs
}

type CoreConcepts struct {
	ConfigMaps        *ConceptFuncs
	Pods              *ConceptFuncs
	Rcs               *ConceptFuncs
	PersistentVolumes *ConceptFuncs
	Deployments       *ConceptFuncs
}

type ExtConcepts struct {
	Deployments *ConceptFuncs
	ReplicaSets *ConceptFuncs
}

func BuildConceptClients(rc func() *restclient.RESTClient, scheme *runtime.Scheme, ns string) *Concepts {

	//TODO investigate how does kubernetes handle pluralizing etc
	c := &Concepts{
		SystemUsers: newConceptFuncs(rc(), scheme, "SystemUser", "SystemUsers", ns),
		Clusters:    newConceptFuncs(rc(), scheme, "Cluster", "Clusters", ns),
		Counters:    newConceptFuncs(rc(), scheme, "Counter", "Counters", ns),
		Sitepods:    newConceptFuncs(rc(), scheme, "Sitepod", "Sitepods", ns),
		Services:    newConceptFuncs(rc(), scheme, "Serviceinstance", "Serviceinstances", ns),
	}

	return c
}

func BuildCoreConceptClients(rc func() *restclient.RESTClient, scheme *runtime.Scheme, ns string) *CoreConcepts {

	//TODO investigate how does kubernetes handle pluralizing etc
	c := &CoreConcepts{
		ConfigMaps:        newConceptFuncs(rc(), scheme, "ConfigMap", "ConfigMaps", ns),
		Rcs:               newConceptFuncs(rc(), scheme, "ReplicationController", "ReplicationControllers", ns),
		Pods:              newConceptFuncs(rc(), scheme, "Pod", "Pods", ns),
		PersistentVolumes: newConceptFuncs(rc(), scheme, "PersistentVolume", "PersistentVolumes", ""),
	}

	return c
}

func BuildExtConceptClients(rc func() *restclient.RESTClient, scheme *runtime.Scheme, ns string) *ExtConcepts {

	//TODO investigate how does kubernetes handle pluralizing etc
	c := &ExtConcepts{
		Deployments: newConceptFuncs(rc(), scheme, "Deployment", "Deployments", ns),
		ReplicaSets: newConceptFuncs(rc(), scheme, "ReplicaSet", "ReplicaSets", ns),
	}

	return c
}

func newConceptFuncs(rc *restclient.RESTClient, scheme *runtime.Scheme, kind string, resource string, ns string) *ConceptFuncs {

	funcs := &ConceptFuncs{}
	funcs.KindSingle = func() string {
		return kind
	}

	funcs.Get = func() *restclient.Request {
		if ns == "" {
			return rc.Get().Resource(resource)
		} else {
			return rc.Get().Resource(resource).Namespace(ns)
		}
	}

	funcs.GetWithLabels = func(s labels.Selector) (runtime.Object, error) {
		var prc *restclient.Request
		if ns == "" {
			prc = rc.Get().Resource(resource).LabelsSelectorParam(s)
		} else {
			prc = rc.Get().Resource(resource).Namespace(ns).LabelsSelectorParam(s)
		}

		rObj, err := prc.Do().Get()
		return rObj, err
	}

	funcs.Post = func() *restclient.Request {
		if ns == "" {
			return rc.Post().Resource(resource)
		} else {
			return rc.Post().Resource(resource).Namespace(ns)
		}
	}

	funcs.Put = func() *restclient.Request {
		if ns == "" {
			return rc.Put().Resource(resource)
		} else {
			return rc.Put().Resource(resource).Namespace(ns)
		}
	}

	funcs.Delete = func() *restclient.Request {
		if ns == "" {
			return rc.Delete().Resource(resource)
		} else {
			return rc.Delete().Resource(resource).Namespace(ns)
		}
	}

	funcs.Getter = func(name string) (runtime.Object, error) {
		item, err := funcs.Get().Name(name).Do().Get()
		if err != nil {
			return nil, err
		}
		return item, nil
	}

	funcs.Updater = func(rObj runtime.Object) (runtime.Object, error) {

		accessor, err := meta.Accessor(rObj)
		if err != nil {
			return nil, err
		}
		uid := accessor.GetUID()
		glog.Errorf("Updating %s", string(uid))
		if len(string(uid)) > 0 {
			rName := accessor.GetName()
			rObj, err = funcs.Put().Name(rName).Body(rObj).Do().Get()
			if err != nil {
				glog.Errorf("Failed updating %s", string(uid))
				return nil, err
			}
			return rObj, nil
		} else {
			return funcs.Adder(rObj)
		}
	}

	funcs.Deleter = func(rObj runtime.Object) error {
		accessor, err := meta.Accessor(rObj)
		if err != nil {
			return err
		}
		err = funcs.Delete().Name(accessor.GetName()).Do().Error()
		return err
	}

	funcs.Adder = func(rObj runtime.Object) (runtime.Object, error) {

		result := funcs.Post().Body(rObj).Do()

		if err := result.Error(); err != nil {
			return nil, err
		}

		return result.Get()
	}

	funcs.ListWatcher = func(fieldSelector fields.Selector) cache.ListerWatcher {
		//TODO: dont generate parameter codec each time
		pc := runtime.NewParameterCodec(scheme)
		return api.NewListWatchFromClient(rc, resource, ns, fieldSelector, pc)
	}

	funcs.Type = func() runtime.Object {
		obj, err := scheme.New(rc.APIVersion().WithKind(kind))
		if err != nil {
			panic(err)
		}
		return obj
	}

	return funcs

}
