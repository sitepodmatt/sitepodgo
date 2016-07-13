package api

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

// ListFunc knows how to list resources
type ListFunc func(options api.ListOptions) (runtime.Object, error)

// WatchFunc knows how to watch resources
type WatchFunc func(options api.ListOptions) (watch.Interface, error)

// ListWatch knows how to list and watch a set of apiserver resources.  It satisfies the ListerWatcher interface.
// It is a convenience function for users of NewReflector, etc.
// ListFunc and WatchFunc must not be nil
type ListWatch struct {
	ListFunc  ListFunc
	WatchFunc WatchFunc
}

// Getter interface knows how to access Get method from RESTClient.
type Getter interface {
	Get() *restclient.Request
}

// NewListWatchFromClient creates a new ListWatch from the specified client, resource, namespace and field selector.
func NewListWatchFromClient(c Getter, resource string, namespace string, fieldSelector fields.Selector, pc runtime.ParameterCodec) *ListWatch {
	listFunc := func(options api.ListOptions) (runtime.Object, error) {
		return c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, pc).
			FieldsSelectorParam(fieldSelector).
			Do().
			Get()
	}
	watchFunc := func(options api.ListOptions) (watch.Interface, error) {
		return c.Get().
			Prefix("watch").
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, pc).
			FieldsSelectorParam(fieldSelector).
			Watch()
	}
	return &ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func timeoutFromListOptions(options api.ListOptions) time.Duration {
	if options.TimeoutSeconds != nil {
		return time.Duration(*options.TimeoutSeconds) * time.Second
	}
	return 0
}

// List a set of apiserver resources
func (lw *ListWatch) List(options api.ListOptions) (runtime.Object, error) {
	return lw.ListFunc(options)
}

// Watch a set of apiserver resources
func (lw *ListWatch) Watch(options api.ListOptions) (watch.Interface, error) {
	return lw.WatchFunc(options)
}
