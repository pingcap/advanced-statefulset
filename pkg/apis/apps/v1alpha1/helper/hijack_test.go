package helper

import (
	"context"
	"testing"
	"time"

	asapps "github.com/cofyc/advanced-statefulset/pkg/apis/apps/v1alpha1"
	asfake "github.com/cofyc/advanced-statefulset/pkg/client/clientset/versioned/fake"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

func verifyObjectRecieve(t *testing.T, ch <-chan *v1.StatefulSet) {
	select {
	case obj := <-ch:
		t.Logf("received an object %T: %q", obj, obj.Name)
	case <-time.After(time.Second * 3):
		t.Errorf("waiting for event timed out")
	}
}

func TestSharedInformerFactory(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	asClient := asfake.NewSimpleClientset()
	hijackClient := NewHijackClient(kubeClient, asClient)
	resyncDuration := time.Second
	kubeInformerFactory := informers.NewSharedInformerFactory(hijackClient, resyncDuration)

	ch := make(chan *v1.StatefulSet)
	kubeInformerFactory.Apps().V1().StatefulSets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sts, ok := obj.(*v1.StatefulSet)
			if !ok {
				klog.Errorf("unexpect object %T: %v received", obj, obj)
				return
			}
			ch <- sts
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kubeInformerFactory.Start(ctx.Done())
	for v, synced := range kubeInformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			t.Fatalf("error syncing informer for %v", v)
		}
	}

	testObj := &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sts",
		},
	}

	testAsObj := &asapps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "as-sts",
		},
	}

	// able to receive event via hijackClient
	_, err := hijackClient.AppsV1().StatefulSets("default").Create(testObj)
	if err != nil {
		t.Fatal(err)
	}
	verifyObjectRecieve(t, ch)

	// able to receive event via asClient
	_, err = asClient.AppsV1alpha1().StatefulSets("default").Create(testAsObj)
	if err != nil {
		t.Fatal(err)
	}
	verifyObjectRecieve(t, ch)
}
