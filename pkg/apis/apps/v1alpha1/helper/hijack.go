package helper

import (
	"encoding/json"

	pcv1alpha1 "github.com/cofyc/advanced-statefulset/pkg/apis/apps/v1alpha1"
	pcclientset "github.com/cofyc/advanced-statefulset/pkg/client/clientset/versioned"
	appsv1alpha1 "github.com/cofyc/advanced-statefulset/pkg/client/clientset/versioned/typed/apps/v1alpha1"
	informersexternal "github.com/cofyc/advanced-statefulset/pkg/client/informers/externalversions"
	informersexternalapps "github.com/cofyc/advanced-statefulset/pkg/client/informers/externalversions/apps"
	informersexternalappsv1alpha1 "github.com/cofyc/advanced-statefulset/pkg/client/informers/externalversions/apps/v1alpha1"
	pclistersappsv1alpha1 "github.com/cofyc/advanced-statefulset/pkg/client/listers/apps/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	apps "k8s.io/client-go/informers/apps"
	informersappsv1 "k8s.io/client-go/informers/apps/v1"
	clientset "k8s.io/client-go/kubernetes"
	clientsetappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	listersappsv1 "k8s.io/client-go/listers/apps/v1"
)

// hijackClient is a special Kubernetes client which hijack statefulset API requests.
type hijackClient struct {
	clientset.Interface
	PingCAPInterface pcclientset.Interface
}

var _ clientset.Interface = &hijackClient{}

func (c hijackClient) AppsV1() clientsetappsv1.AppsV1Interface {
	return hijackAppsV1Client{c.Interface.AppsV1(), c.PingCAPInterface.AppsV1alpha1()}
}

// NewHijackClient creates a new hijacked Kubernetes interface.
func NewHijackClient(client clientset.Interface, pcclient pcclientset.Interface) clientset.Interface {
	return &hijackClient{client, pcclient}
}

type hijackAppsV1Client struct {
	clientsetappsv1.AppsV1Interface
	pingcapV1alpha1Client appsv1alpha1.AppsV1alpha1Interface
}

var _ clientsetappsv1.AppsV1Interface = &hijackAppsV1Client{}

func (c hijackAppsV1Client) StatefulSets(namespace string) clientsetappsv1.StatefulSetInterface {
	return &hijackStatefulset{c.pingcapV1alpha1Client.StatefulSets(namespace)}
}

type hijackStatefulset struct {
	appsv1alpha1.StatefulSetInterface
}

var _ clientsetappsv1.StatefulSetInterface = &hijackStatefulset{}

func (s *hijackStatefulset) Create(sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	pcsts, err := FromBuiltinStatefulSet(sts)
	if err != nil {
		return nil, err
	}
	pcv1alpha1.SetObjectDefaults_StatefulSet(pcsts) // required if defaulting is not enabled in kube-apiserver
	pcsts, err = s.StatefulSetInterface.Create(pcsts)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func (s *hijackStatefulset) Update(sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
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

func (s *hijackStatefulset) UpdateStatus(sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
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

func (s *hijackStatefulset) Get(name string, options metav1.GetOptions) (*appsv1.StatefulSet, error) {
	pcsts, err := s.StatefulSetInterface.Get(name, options)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func (s *hijackStatefulset) List(opts metav1.ListOptions) (*appsv1.StatefulSetList, error) {
	list, err := s.StatefulSetInterface.List(opts)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStetefulsetList(list)
}

func (s *hijackStatefulset) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *appsv1.StatefulSet, err error) {
	pcsts, err := s.StatefulSetInterface.Patch(name, pt, data, subresources...)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

// hijackSharedInformerFactory hijacks StatefulSet V1 interface of informers.SharedInformerFactory.
type hijackSharedInformerFactory struct {
	informers.SharedInformerFactory
	externalSharedInformerFactory informersexternal.SharedInformerFactory
}

var _ informers.SharedInformerFactory = &hijackSharedInformerFactory{}

func NewHijackSharedInformerFactory(informerFactory informers.SharedInformerFactory, externalInformerFactory informersexternal.SharedInformerFactory) informers.SharedInformerFactory {
	return &hijackSharedInformerFactory{informerFactory, externalInformerFactory}
}

func (f *hijackSharedInformerFactory) Apps() apps.Interface {
	return &hijackAppsInterface{f.SharedInformerFactory.Apps(), f.externalSharedInformerFactory.Apps()}
}

type hijackAppsInterface struct {
	apps.Interface
	externalInterface informersexternalapps.Interface
}

var _ apps.Interface = &hijackAppsInterface{}

func (i *hijackAppsInterface) V1() informersappsv1.Interface {
	return &hijackInformersAppsV1Interface{i.Interface.V1(), i.externalInterface.V1alpha1()}
}

type hijackInformersAppsV1Interface struct {
	informersappsv1.Interface
	externalInterface informersexternalappsv1alpha1.Interface
}

var _ informersappsv1.Interface = &hijackInformersAppsV1Interface{}

func (v *hijackInformersAppsV1Interface) StatefulSets() informersappsv1.StatefulSetInformer {
	return &hijackStatefulSetInformer{v.externalInterface.StatefulSets()}
}

type hijackStatefulSetInformer struct {
	informersexternalappsv1alpha1.StatefulSetInformer
}

var _ informersappsv1.StatefulSetInformer = &hijackStatefulSetInformer{}

func (f *hijackStatefulSetInformer) Lister() listersappsv1.StatefulSetLister {
	return &hijackStatefulSetLister{f.StatefulSetInformer.Lister()}
}

type hijackStatefulSetLister struct {
	pclistersappsv1alpha1.StatefulSetLister
}

var _ listersappsv1.StatefulSetLister = &hijackStatefulSetLister{}

func (l *hijackStatefulSetLister) List(selector labels.Selector) (ret []*appsv1.StatefulSet, err error) {
	items, err := l.StatefulSetLister.List(selector)
	if err != nil {
		return nil, err
	}
	return ToMultiBuiltinStatefulSet(items)
}

func (l *hijackStatefulSetLister) StatefulSets(namespace string) listersappsv1.StatefulSetNamespaceLister {
	return &hijackStatefulSetNamespaceLister{l.StatefulSetLister.StatefulSets(namespace)}
}

func (l *hijackStatefulSetLister) GetPodStatefulSets(pod *v1.Pod) ([]*appsv1.StatefulSet, error) {
	items, err := l.StatefulSetLister.GetPodStatefulSets(pod)
	if err != nil {
		return nil, err
	}
	return ToMultiBuiltinStatefulSet(items)
}

type hijackStatefulSetNamespaceLister struct {
	pclistersappsv1alpha1.StatefulSetNamespaceLister
}

var _ listersappsv1.StatefulSetNamespaceLister = &hijackStatefulSetNamespaceLister{}

func (l *hijackStatefulSetNamespaceLister) List(selector labels.Selector) (ret []*appsv1.StatefulSet, err error) {
	items, err := l.StatefulSetNamespaceLister.List(selector)
	if err != nil {
		return nil, err
	}
	return ToMultiBuiltinStatefulSet(items)
}

func (l *hijackStatefulSetNamespaceLister) Get(name string) (*appsv1.StatefulSet, error) {
	pcsts, err := l.StatefulSetNamespaceLister.Get(name)
	if err != nil {
		return nil, err
	}
	return ToBuiltinStatefulSet(pcsts)
}

func FromBuiltinStatefulSet(sts *appsv1.StatefulSet) (*pcv1alpha1.StatefulSet, error) {
	data, err := json.Marshal(sts)
	if err != nil {
		return nil, err
	}
	newSet := &pcv1alpha1.StatefulSet{}
	err = json.Unmarshal(data, newSet)
	return newSet, err
}

func ToMultiBuiltinStatefulSet(pcstss []*pcv1alpha1.StatefulSet) ([]*appsv1.StatefulSet, error) {
	stss := make([]*appsv1.StatefulSet, 0)
	for _, pcsts := range pcstss {
		sts, err := ToBuiltinStatefulSet(pcsts)
		if err != nil {
			return nil, err
		}
		stss = append(stss, sts)
	}
	return stss, nil
}

func ToBuiltinStatefulSet(sts *pcv1alpha1.StatefulSet) (*appsv1.StatefulSet, error) {
	data, err := json.Marshal(sts)
	if err != nil {
		return nil, err
	}
	newSet := &appsv1.StatefulSet{}
	err = json.Unmarshal(data, newSet)
	return newSet, err
}

func ToBuiltinStetefulsetList(stsList *pcv1alpha1.StatefulSetList) (*appsv1.StatefulSetList, error) {
	data, err := json.Marshal(stsList)
	if err != nil {
		return nil, err
	}
	newSet := &appsv1.StatefulSetList{}
	err = json.Unmarshal(data, newSet)
	return newSet, err
}
