package helper

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	asapps "github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1alpha1"
	asfake "github.com/pingcap/advanced-statefulset/pkg/client/clientset/versioned/fake"
	v1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

var (
	ns = "default"

	testObj = &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sts",
		},
	}

	testAsObj = &asapps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "as-sts",
		},
	}
)

func verifyObjectRecieve(t *testing.T, ch <-chan *v1.StatefulSet) {
	select {
	case obj := <-ch:
		t.Logf("received an object %T: %q", obj, obj.Name)
	case <-time.After(time.Second * 3):
		t.Errorf("waiting for event timed out")
	}
}

func TestHijackClientSet(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	asClient := asfake.NewSimpleClientset()
	hijackClient := NewHijackClient(kubeClient, asClient)

	appsv1stscli := hijackClient.AppsV1().StatefulSets(ns)
	newObj, err := appsv1stscli.Create(testObj)
	if err != nil {
		t.Fatal(err)
	}
	getObj, err := appsv1stscli.Get(testObj.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(newObj, getObj); diff != "" {
		t.Errorf("unexpected result (-want, +got): %s", diff)
	}
	_, err = appsv1stscli.Create(testObj)
	if !apierrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}
}

func TestSharedInformerFactory(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	asClient := asfake.NewSimpleClientset()
	hijackClient := NewHijackClient(kubeClient, asClient)
	resyncDuration := time.Second
	kubeInformerFactory := informers.NewSharedInformerFactory(hijackClient, resyncDuration)

	ch := make(chan *v1.StatefulSet)
	stsInformer := kubeInformerFactory.Apps().V1().StatefulSets().Informer()
	stsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sts, ok := obj.(*v1.StatefulSet)
			if !ok {
				klog.Errorf("unexpect object %T: %v received", obj, obj)
				return
			}
			ch <- sts
		},
	})
	stsLister := kubeInformerFactory.Apps().V1().StatefulSets().Lister()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kubeInformerFactory.Start(ctx.Done())
	for v, synced := range kubeInformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			t.Fatalf("error syncing informer for %v", v)
		}
	}

	t.Log("able to receive event via hijackClient")
	wantSts, err := hijackClient.AppsV1().StatefulSets(ns).Create(testObj)
	if err != nil {
		t.Fatal(err)
	}
	verifyObjectRecieve(t, ch)

	t.Log("able to receive event via asClient")
	_, err = asClient.AppsV1alpha1().StatefulSets(ns).Create(testAsObj)
	if err != nil {
		t.Fatal(err)
	}
	verifyObjectRecieve(t, ch)

	t.Log("able to fetch via lister")
	sts, err := stsLister.StatefulSets(ns).Get("sts")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantSts, sts); diff != "" {
		t.Errorf("unexpected result (-want, +got): %s", diff)
	}

	t.Log("not found error if sts does not exist")
	_, err = stsLister.StatefulSets(ns).Get("does-not-exist")
	if !apierrors.IsNotFound(err) {
		t.Fatal(err)
	}
}
