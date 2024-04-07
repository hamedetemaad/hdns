package v1alpha1

import (
	"github.com/hamedetemaad/hdns-operator/pkg/dnsblock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeGroupVersion = schema.GroupVersion{
		Group:   dnsblock.GroupName,
		Version: dnsblock.V1alpha1,
	}
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&DNSBlock{},
		&DNSBlockList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
}
