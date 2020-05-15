// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package helper

import (
	"encoding/json"
	"sync"

	asv1 "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	asclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	asclientsetv1 "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned/typed/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	clientsetappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// hijackClient is a special Kubernetes client which hijack statefulset API requests.
type hijackClient struct {
	clientset.Interface
	asInterface asclientset.Interface
}

var _ clientset.Interface = &hijackClient{}

func (c hijackClient) AppsV1() clientsetappsv1.AppsV1Interface {
	return hijackAppsV1Client{c.Interface.AppsV1(), c.asInterface.AppsV1()}
}

// NewHijackClient creates a new hijacked Kubernetes interface.
func NewHijackClient(client clientset.Interface, pcclient asclientset.Interface) clientset.Interface {
	return &hijackClient{client, pcclient}
}

type hijackAppsV1Client struct {
	clientsetappsv1.AppsV1Interface
	pingcapV1Client asclientsetv1.AppsV1Interface
}

var _ clientsetappsv1.AppsV1Interface = &hijackAppsV1Client{}

func (c hijackAppsV1Client) StatefulSets(namespace string) clientsetappsv1.StatefulSetInterface {
	return &hijackStatefulSet{c.pingcapV1Client.StatefulSets(namespace)}
}

type hijackStatefulSet struct {
	asclientsetv1.StatefulSetInterface
}

var _ clientsetappsv1.StatefulSetInterface = &hijackStatefulSet{}

func (s *hijackStatefulSet) Create(sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	pcsts, err := FromBuiltinStatefulSet(sts)
	if err != nil {
		return nil, err
	}
	asv1.SetObjectDefaults_StatefulSet(pcsts) // required if defaulting is not enabled in kube-apiserver
	pcsts, err = s.StatefulSetInterface.Create(pcsts)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func (s *hijackStatefulSet) Update(sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	pcsts, err := FromBuiltinStatefulSet(sts)
	if err != nil {
		return nil, err
	}
	asv1.SetObjectDefaults_StatefulSet(pcsts) // required if defaulting is not enabled in kube-apiserver
	pcsts, err = s.StatefulSetInterface.Update(pcsts)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func (s *hijackStatefulSet) UpdateStatus(sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	pcsts, err := FromBuiltinStatefulSet(sts)
	if err != nil {
		return nil, err
	}
	pcsts, err = s.StatefulSetInterface.UpdateStatus(pcsts)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func (s *hijackStatefulSet) Get(name string, options metav1.GetOptions) (*appsv1.StatefulSet, error) {
	pcsts, err := s.StatefulSetInterface.Get(name, options)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func (s *hijackStatefulSet) List(opts metav1.ListOptions) (*appsv1.StatefulSetList, error) {
	list, err := s.StatefulSetInterface.List(opts)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStetefulsetList(list)
}

func (s *hijackStatefulSet) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	watch, err := s.StatefulSetInterface.Watch(opts)
	if err != nil {
		return nil, err
	}
	return newHijackWatch(watch), nil
}

type hijackWatch struct {
	sync.Mutex
	source  watch.Interface
	result  chan watch.Event
	stopped bool
}

func newHijackWatch(source watch.Interface) watch.Interface {
	w := &hijackWatch{
		source: source,
		result: make(chan watch.Event),
	}
	go w.receive()
	return w
}

func (w *hijackWatch) Stop() {
	w.Lock()
	defer w.Unlock()
	if !w.stopped {
		w.stopped = true
		w.source.Stop()
	}
}

func (w *hijackWatch) receive() {
	defer close(w.result)
	defer w.Stop()
	defer utilruntime.HandleCrash()
	for {
		select {
		case event, ok := <-w.source.ResultChan():
			if !ok {
				return
			}
			asts, ok := event.Object.(*asv1.StatefulSet)
			if !ok {
				panic("unreachable")
			}
			sts, err := ToBuiltinStatefulSet(asts)
			if err != nil {
				panic(err)
			}
			w.result <- watch.Event{
				Type:   event.Type,
				Object: sts,
			}
		}
	}
}

func (w *hijackWatch) ResultChan() <-chan watch.Event {
	w.Lock()
	defer w.Unlock()
	return w.result
}

func (s *hijackStatefulSet) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *appsv1.StatefulSet, err error) {
	pcsts, err := s.StatefulSetInterface.Patch(name, pt, data, subresources...)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func FromBuiltinStatefulSet(sts *appsv1.StatefulSet) (*asv1.StatefulSet, error) {
	data, err := json.Marshal(sts)
	if err != nil {
		return nil, err
	}
	newSet := &asv1.StatefulSet{}
	err = json.Unmarshal(data, newSet)
	if err != nil {
		return nil, err
	}
	newSet.TypeMeta.APIVersion = asv1.SchemeGroupVersion.String()
	return newSet, nil
}

func ToBuiltinStatefulSet(sts *asv1.StatefulSet) (*appsv1.StatefulSet, error) {
	data, err := json.Marshal(sts)
	if err != nil {
		return nil, err
	}
	newSet := &appsv1.StatefulSet{}
	err = json.Unmarshal(data, newSet)
	if err != nil {
		return nil, err
	}
	newSet.TypeMeta.APIVersion = appsv1.SchemeGroupVersion.String()
	return newSet, nil
}

func ToBuiltinStetefulsetList(stsList *asv1.StatefulSetList) (*appsv1.StatefulSetList, error) {
	data, err := json.Marshal(stsList)
	if err != nil {
		return nil, err
	}
	newList := &appsv1.StatefulSetList{}
	err = json.Unmarshal(data, newList)
	if err != nil {
		return nil, err
	}
	newList.TypeMeta.APIVersion = appsv1.SchemeGroupVersion.String()
	for i, sts := range newList.Items {
		sts.TypeMeta.APIVersion = appsv1.SchemeGroupVersion.String()
		newList.Items[i] = sts
	}
	return newList, nil
}
