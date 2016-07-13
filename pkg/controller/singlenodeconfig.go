package controller

import (
	"time"

	"sitepod.io/sitepod/pkg/api/v1"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/meta"
	k8s_v1 "k8s.io/kubernetes/pkg/api/v1"
	ext_api "k8s.io/kubernetes/pkg/apis/extensions"
	ext_v1 "k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime/serializer"
)

type SingleNodeConfig struct {
	SitepodInformer    framework.SharedIndexInformer
	PodInformer        framework.SharedIndexInformer
	DeploymentInformer framework.SharedIndexInformer
	ServicesInformer   framework.SharedIndexInformer
	PvInformer         framework.SharedIndexInformer
	ConfigMapInformer  framework.SharedIndexInformer
	SystemUserInformer framework.SharedIndexInformer
	CoreConcepts       *v1.CoreConcepts
	ExtConcepts        *v1.ExtConcepts
	Concepts           *v1.Concepts
}

//TODO: inject as host configuration
var (
	hostPath = "http://127.0.0.1:8080"
	apiPath  = "apis"
)

func DefaultConfig() *SingleNodeConfig {

	apiPath, _, externalGV := v1.AddToScheme(api.Scheme)
	k8s_v1.AddToScheme(api.Scheme)
	api.AddToScheme(api.Scheme)
	ext_v1.AddToScheme(api.Scheme)
	ext_api.AddToScheme(api.Scheme)

	schemeSerializer := serializer.NewCodecFactory(api.Scheme)

	restClientFactory := func() *restclient.RESTClient {
		restConfig := &restclient.Config{
			Host:    hostPath,
			APIPath: apiPath,
			ContentConfig: restclient.ContentConfig{
				GroupVersion:         externalGV,
				NegotiatedSerializer: schemeSerializer,
			},
		}

		restClient, err := restclient.RESTClientFor(restConfig)
		if err != nil {
			panic(err)
		}
		return restClient
	}

	coreRestClientFactory := func() *restclient.RESTClient {
		coreRestConfig := &restclient.Config{
			Host:    hostPath,
			APIPath: "api",
			ContentConfig: restclient.ContentConfig{
				GroupVersion:         &k8s_v1.SchemeGroupVersion,
				NegotiatedSerializer: api.Codecs,
			},
		}

		coreRestClient, err := restclient.RESTClientFor(coreRestConfig)
		if err != nil {
			panic(err)
		}
		return coreRestClient
	}

	extRestClientFactory := func() *restclient.RESTClient {
		coreRestConfig := &restclient.Config{
			Host:    hostPath,
			APIPath: "apis",
			ContentConfig: restclient.ContentConfig{
				GroupVersion:         &ext_v1.SchemeGroupVersion,
				NegotiatedSerializer: api.Codecs,
			},
		}

		coreRestClient, err := restclient.RESTClientFor(coreRestConfig)
		if err != nil {
			panic(err)
		}
		return coreRestClient
	}

	resyncPeriod := 5 * time.Minute
	config := &SingleNodeConfig{}

	var coreConcepts = v1.BuildCoreConceptClients(coreRestClientFactory, api.Scheme, "default")
	var extConcepts = v1.BuildExtConceptClients(extRestClientFactory, api.Scheme, "default")
	var concepts = v1.BuildConceptClients(restClientFactory, api.Scheme, "default")

	indexers := make(cache.Indexers)
	indexers["sitepod"] = func(obj interface{}) ([]string, error) {
		accessor, _ := meta.Accessor(obj)
		labels := accessor.GetLabels()
		if _, ok := labels["sitepod"]; ok {
			return []string{labels["sitepod"]}, nil
		} else {
			return []string{}, nil
		}
	}
	indexers["uid"] = func(obj interface{}) ([]string, error) {
		accessor, _ := meta.Accessor(obj)
		return []string{string(accessor.GetUID())}, nil
	}

	config.SitepodInformer = framework.NewSharedIndexInformer(
		concepts.Sitepods.ListWatcher(nil),
		&v1.Sitepod{},
		resyncPeriod,
		indexers,
	)
	config.ServicesInformer = framework.NewSharedIndexInformer(
		concepts.Services.ListWatcher(nil),
		&v1.Serviceinstance{},
		resyncPeriod,
		indexers,
	)

	config.PvInformer = framework.NewSharedIndexInformer(
		coreConcepts.PersistentVolumes.ListWatcher(nil),
		&api.PersistentVolume{},
		resyncPeriod,
		indexers,
	)

	config.PodInformer = framework.NewSharedIndexInformer(
		coreConcepts.Pods.ListWatcher(nil),
		&api.Pod{},
		resyncPeriod,
		indexers,
	)

	config.ConfigMapInformer = framework.NewSharedIndexInformer(
		coreConcepts.ConfigMaps.ListWatcher(nil),
		&api.ConfigMap{},
		resyncPeriod,
		indexers,
	)

	config.SystemUserInformer = framework.NewSharedIndexInformer(
		concepts.SystemUsers.ListWatcher(nil),
		&v1.SystemUser{},
		resyncPeriod,
		indexers,
	)

	config.DeploymentInformer = framework.NewSharedIndexInformer(
		extConcepts.Deployments.ListWatcher(nil),
		&ext_api.Deployment{},
		resyncPeriod,
		indexers)

	config.Concepts = concepts
	config.CoreConcepts = coreConcepts
	config.ExtConcepts = extConcepts

	return config
}
