package helper

import (
	"encoding/json"

	pcv1alpha1 "github.com/cofyc/advanced-statefulset/pkg/apis/pingcap/v1alpha1"
	pcclientset "github.com/cofyc/advanced-statefulset/pkg/client/clientset/versioned"
	pingcapv1alpha1 "github.com/cofyc/advanced-statefulset/pkg/client/clientset/versioned/typed/pingcap/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// HijackClient is a special Kubernetes client which hijack statefulset API requests.
type HijackClient struct {
	clientset.Interface
	PingCAPInterface pcclientset.Interface
}

func (c HijackClient) AppsV1() appsv1.AppsV1Interface {
	return hijackAppsV1Client{c.Interface.AppsV1(), c.PingCAPInterface.PingcapV1alpha1()}
}

var _ clientset.Interface = &HijackClient{}

type hijackAppsV1Client struct {
	appsv1.AppsV1Interface
	pingcapV1alpha1Client pingcapv1alpha1.PingcapV1alpha1Interface
}

func (c hijackAppsV1Client) StatefulSets(namespace string) appsv1.StatefulSetInterface {
	return &hijackStatefulset{c.pingcapV1alpha1Client.StatefulSets(namespace)}
}

var _ appsv1.AppsV1Interface = &hijackAppsV1Client{}

type hijackStatefulset struct {
	pingcapv1alpha1.StatefulSetInterface
}

var _ appsv1.StatefulSetInterface = &hijackStatefulset{}

func (s *hijackStatefulset) Create(sts *v1.StatefulSet) (*v1.StatefulSet, error) {
	pcsts, err := FromBuiltinStatefulSet(sts)
	if err != nil {
		return nil, err
	}
	pcsts, err = s.StatefulSetInterface.Create(pcsts)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func (s *hijackStatefulset) Update(sts *v1.StatefulSet) (*v1.StatefulSet, error) {
	pcsts, err := FromBuiltinStatefulSet(sts)
	if err != nil {
		return nil, err
	}
	pcsts, err = s.StatefulSetInterface.Update(pcsts)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func (s *hijackStatefulset) UpdateStatus(sts *v1.StatefulSet) (*v1.StatefulSet, error) {
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

func (s *hijackStatefulset) Get(name string, options metav1.GetOptions) (*v1.StatefulSet, error) {
	pcsts, err := s.StatefulSetInterface.Get(name, options)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func (s *hijackStatefulset) List(opts metav1.ListOptions) (*v1.StatefulSetList, error) {
	list, err := s.StatefulSetInterface.List(opts)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStetefulsetList(list)
}

func (s *hijackStatefulset) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.StatefulSet, err error) {
	pcsts, err := s.StatefulSetInterface.Patch(name, pt, data, subresources...)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func FromBuiltinStatefulSet(sts *v1.StatefulSet) (*pcv1alpha1.StatefulSet, error) {
	data, err := json.Marshal(sts)
	if err != nil {
		return nil, err
	}
	newSet := &pcv1alpha1.StatefulSet{}
	err = json.Unmarshal(data, newSet)
	return newSet, err
}

func ToBuiltinStatefulSet(sts *pcv1alpha1.StatefulSet) (*v1.StatefulSet, error) {
	data, err := json.Marshal(sts)
	if err != nil {
		return nil, err
	}
	newSet := &v1.StatefulSet{}
	err = json.Unmarshal(data, newSet)
	return newSet, err
}

func ToBuiltinStetefulsetList(stsList *pcv1alpha1.StatefulSetList) (*v1.StatefulSetList, error) {
	data, err := json.Marshal(stsList)
	if err != nil {
		return nil, err
	}
	newSet := &v1.StatefulSetList{}
	err = json.Unmarshal(data, newSet)
	return newSet, err
}
