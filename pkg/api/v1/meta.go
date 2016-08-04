package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
)

// CRITICAL - Decoder is dispatched on v1.ObjectMeta so we have to be careful

type ObjectMeta v1.ObjectMeta

type ListMeta unversioned.ListMeta
