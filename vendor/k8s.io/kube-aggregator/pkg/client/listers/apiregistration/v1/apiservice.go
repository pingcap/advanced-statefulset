/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

// APIServiceLister helps list APIServices.
// All objects returned here must be treated as read-only.
type APIServiceLister interface {
	// List lists all APIServices in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.APIService, err error)
	// Get retrieves the APIService from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.APIService, error)
	APIServiceListerExpansion
}

// aPIServiceLister implements the APIServiceLister interface.
type aPIServiceLister struct {
	indexer cache.Indexer
}

// NewAPIServiceLister returns a new APIServiceLister.
func NewAPIServiceLister(indexer cache.Indexer) APIServiceLister {
	return &aPIServiceLister{indexer: indexer}
}

// List lists all APIServices in the indexer.
func (s *aPIServiceLister) List(selector labels.Selector) (ret []*v1.APIService, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.APIService))
	})
	return ret, err
}

// Get retrieves the APIService from the index for a given name.
func (s *aPIServiceLister) Get(name string) (*v1.APIService, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("apiservice"), name)
	}
	return obj.(*v1.APIService), nil
}
