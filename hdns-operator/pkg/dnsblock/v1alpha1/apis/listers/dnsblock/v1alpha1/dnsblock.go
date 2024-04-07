/* AUTO GENERATED CODE */
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/hamedetemaad/hdns-operator/pkg/dnsblock/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// DNSBlockLister helps list DNSBlocks.
// All objects returned here must be treated as read-only.
type DNSBlockLister interface {
	// List lists all DNSBlocks in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.DNSBlock, err error)
	// DNSBlocks returns an object that can list and get DNSBlocks.
	DNSBlocks(namespace string) DNSBlockNamespaceLister
	DNSBlockListerExpansion
}

// dNSBlockLister implements the DNSBlockLister interface.
type dNSBlockLister struct {
	indexer cache.Indexer
}

// NewDNSBlockLister returns a new DNSBlockLister.
func NewDNSBlockLister(indexer cache.Indexer) DNSBlockLister {
	return &dNSBlockLister{indexer: indexer}
}

// List lists all DNSBlocks in the indexer.
func (s *dNSBlockLister) List(selector labels.Selector) (ret []*v1alpha1.DNSBlock, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.DNSBlock))
	})
	return ret, err
}

// DNSBlocks returns an object that can list and get DNSBlocks.
func (s *dNSBlockLister) DNSBlocks(namespace string) DNSBlockNamespaceLister {
	return dNSBlockNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// DNSBlockNamespaceLister helps list and get DNSBlocks.
// All objects returned here must be treated as read-only.
type DNSBlockNamespaceLister interface {
	// List lists all DNSBlocks in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.DNSBlock, err error)
	// Get retrieves the DNSBlock from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.DNSBlock, error)
	DNSBlockNamespaceListerExpansion
}

// dNSBlockNamespaceLister implements the DNSBlockNamespaceLister
// interface.
type dNSBlockNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all DNSBlocks in the indexer for a given namespace.
func (s dNSBlockNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.DNSBlock, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.DNSBlock))
	})
	return ret, err
}

// Get retrieves the DNSBlock from the indexer for a given namespace and name.
func (s dNSBlockNamespaceLister) Get(name string) (*v1alpha1.DNSBlock, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("dnsblock"), name)
	}
	return obj.(*v1alpha1.DNSBlock), nil
}
