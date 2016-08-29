package v1

import (
	k8s_api "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	k8s_v1 "k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/runtime"
)

const (
	APIPath         string = "apis" // all third party resource use 'apis/'
	Group           string = "stable.sitepod.io"
	ExternalVersion string = "v1"
	InternalVersion string = runtime.APIVersionInternal
)

func AddToScheme(s *runtime.Scheme) (string, *unversioned.GroupVersion, *unversioned.GroupVersion) {

	internalGV := unversioned.GroupVersion{Group: Group, Version: InternalVersion}
	externalGV := unversioned.GroupVersion{Group: Group, Version: ExternalVersion}

	// TODO: Use seperate types for external and internal
	s.AddKnownTypes(internalGV, &SystemUser{})
	s.AddKnownTypes(externalGV, &SystemUser{})
	s.AddKnownTypes(internalGV, &SystemUserList{})
	s.AddKnownTypes(externalGV, &SystemUserList{})

	s.AddKnownTypes(internalGV, &Sitepod{})
	s.AddKnownTypes(externalGV, &Sitepod{})
	s.AddKnownTypes(internalGV, &SitepodList{})
	s.AddKnownTypes(externalGV, &SitepodList{})

	s.AddKnownTypes(internalGV, &Appcomponent{})
	s.AddKnownTypes(externalGV, &Appcomponent{})
	s.AddKnownTypes(internalGV, &AppcomponentList{})
	s.AddKnownTypes(externalGV, &AppcomponentList{})

	s.AddKnownTypes(internalGV, &Cluster{})
	s.AddKnownTypes(externalGV, &Cluster{})
	s.AddKnownTypes(internalGV, &ClusterList{})
	s.AddKnownTypes(externalGV, &ClusterList{})

	s.AddKnownTypes(internalGV, &Website{})
	s.AddKnownTypes(externalGV, &Website{})
	s.AddKnownTypes(internalGV, &WebsiteList{})
	s.AddKnownTypes(externalGV, &WebsiteList{})

	s.AddKnownTypes(internalGV, &Podtask{})
	s.AddKnownTypes(externalGV, &Podtask{})
	s.AddKnownTypes(internalGV, &PodtaskList{})
	s.AddKnownTypes(externalGV, &PodtaskList{})
	//TODO k8s reflector uses api.ListOptions, can we escape this
	//dependency without rewriting?
	s.AddKnownTypes(externalGV, &k8s_v1.ListOptions{})
	s.AddKnownTypes(internalGV, &k8s_api.ListOptions{})

	s.AddConversionFuncs(k8s_v1.Convert_v1_ListOptions_To_api_ListOptions,
		k8s_v1.Convert_api_ListOptions_To_v1_ListOptions)

	return APIPath, &internalGV, &externalGV
}

func DefaultKeyFunc(obj interface{}) (string, error) {
	return cache.MetaNamespaceKeyFunc(obj)
}
