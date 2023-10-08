/*
Copyright 2018 The Kubernetes Authors.

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

package statefulset

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	appsv1 "github.com/pingcap/advanced-statefulset/client/apis/apps/v1"
	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	pcclientset "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned"
	typedappsv1 "github.com/pingcap/advanced-statefulset/client/client/clientset/versioned/typed/apps/v1"
	pcinformers "github.com/pingcap/advanced-statefulset/client/client/informers/externalversions"
	"github.com/pingcap/advanced-statefulset/pkg/controller/statefulset"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

const (
	pollInterval = 100 * time.Millisecond
	pollTimeout  = 60 * time.Second
)

func labelMap() map[string]string {
	return map[string]string{"foo": "bar"}
}

// newService returns a service with a fake name for StatefulSet to be created soon
func newHeadlessService(namespace string) *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "fake-service-name",
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{Port: 80, Name: "http", Protocol: "TCP"},
			},
			Selector: labelMap(),
		},
	}
}

// newSTS returns a StatefulSet with a fake container image
func newSTS(name, namespace string, replicas int) *appsv1.StatefulSet {
	replicasCopy := int32(replicas)
	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: appsv1.StatefulSetSpec{
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            &replicasCopy,
			Selector: &metav1.LabelSelector{
				MatchLabels: labelMap(),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelMap(),
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "fake-name",
							Image: "fakeimage",
							VolumeMounts: []v1.VolumeMount{
								{Name: "datadir", MountPath: "/data/"},
								{Name: "home", MountPath: "/home"},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "datadir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: fmt.Sprintf("/tmp/%v", "datadir"),
								},
							},
						},
						{
							Name: "home",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: fmt.Sprintf("/tmp/%v", "home"),
								},
							},
						},
					},
				},
			},
			ServiceName: "fake-service-name",
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{
				// for volume mount "datadir"
				newStatefulSetPVC("fake-pvc-name"),
			},
		},
	}
	// When using CRD, we must set defaults in client-side.
	appsv1.SetObjectDefaults_StatefulSet(sts)
	return sts
}

func newStatefulSetPVC(name string) v1.PersistentVolumeClaim {
	return v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"volume.alpha.kubernetes.io/storage-class": "anything",
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: *resource.NewQuantity(1, resource.BinarySI),
				},
			},
		},
	}
}

// CloseFunc can be called to cleanup the API server
// copied from `framework.CloseFunc`
type CloseFunc func()

// scSetup sets up necessities for StatefulSet integration test, including master, apiserver, informers, and clientset
func scSetup(t *testing.T) (CloseFunc, *statefulset.StatefulSetController, informers.SharedInformerFactory, clientset.Interface, pcinformers.SharedInformerFactory, pcclientset.Interface) {
	// NOTE(pingcap): already disabled integration
	return nil, nil, nil, nil, nil, nil

	/*master := integrationetcd.StartRealAPIServerOrDie(t)

	var config restclient.Config
	config = *master.Config
	apiExtensionsClient, err := apiextensionsclientset.NewForConfig(&config)
	if err != nil {
		t.Fatalf("error in create clientset: %v", err)
	}
	integrationetcd.CreateTestCRDs(t, apiExtensionsClient, false, getCustomResourceDefinitionData()...)
	clientSet, err := clientset.NewForConfig(&config)
	if err != nil {
		t.Fatalf("error in create clientset: %v", err)
	}
	var pcConfig restclient.Config
	pcConfig = *master.Config
	pcConfig.ContentConfig.ContentType = "application/json" // CRDs does not support protocol buffers
	pcclientSet, err := pcclientset.NewForConfig(&pcConfig)
	if err != nil {
		t.Fatalf("error in create clientset: %v", err)
	}
	resyncPeriod := 12 * time.Hour
	informers := informers.NewSharedInformerFactory(clientset.NewForConfigOrDie(restclient.AddUserAgent(&config, "statefulset-informers")), resyncPeriod)
	pcinformers := pcinformers.NewSharedInformerFactory(pcclientset.NewForConfigOrDie(restclient.AddUserAgent(&pcConfig, "statefulset-informers")), resyncPeriod)

	sc := statefulset.NewStatefulSetController(
		informers.Core().V1().Pods(),
		pcinformers.Apps().V1().StatefulSets(),
		informers.Core().V1().PersistentVolumeClaims(),
		informers.Apps().V1().ControllerRevisions(),
		clientset.NewForConfigOrDie(restclient.AddUserAgent(&config, "statefulset-controller")),
		pcclientset.NewForConfigOrDie(restclient.AddUserAgent(&pcConfig, "statefulset-controller")),
	)

	return master.Cleanup, sc, informers, clientSet, pcinformers, pcclientSet*/
}

// Run STS controller and informers
func runControllerAndInformers(sc *statefulset.StatefulSetController, informers informers.SharedInformerFactory, pcinformers pcinformers.SharedInformerFactory) chan struct{} {
	stopCh := make(chan struct{})
	informers.Start(stopCh)
	pcinformers.Start(stopCh)
	go sc.Run(5, stopCh)
	return stopCh
}

func createHeadlessService(t *testing.T, clientSet clientset.Interface, headlessService *v1.Service) {
	_, err := clientSet.CoreV1().Services(headlessService.Namespace).Create(context.TODO(), headlessService, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed creating headless service: %v", err)
	}
}

func createSTSsPods(t *testing.T, clientSet clientset.Interface, pcclientSet pcclientset.Interface, stss []*appsv1.StatefulSet, pods []*v1.Pod) ([]*appsv1.StatefulSet, []*v1.Pod) {
	var createdSTSs []*appsv1.StatefulSet
	var createdPods []*v1.Pod
	for _, sts := range stss {
		createdSTS, err := pcclientSet.AppsV1().StatefulSets(sts.Namespace).Create(context.TODO(), sts, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("failed to create sts %s: %v", sts.Name, err)
		}
		createdSTSs = append(createdSTSs, createdSTS)
	}
	for _, pod := range pods {
		createdPod, err := clientSet.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("failed to create pod %s: %v", pod.Name, err)
		}
		createdPods = append(createdPods, createdPod)
	}

	return createdSTSs, createdPods
}

// Verify .Status.Replicas is equal to .Spec.Replicas
func waitSTSStable(t *testing.T, pcclientSet pcclientset.Interface, sts *appsv1.StatefulSet) {
	stsClient := pcclientSet.AppsV1().StatefulSets(sts.Namespace)
	desiredGeneration := sts.Generation
	if err := wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		newSTS, err := stsClient.Get(context.TODO(), sts.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return newSTS.Status.Replicas == *newSTS.Spec.Replicas && newSTS.Status.ObservedGeneration >= desiredGeneration, nil
	}); err != nil {
		t.Fatalf("failed to verify .Status.Replicas is equal to .Spec.Replicas for sts %s (got %d, expected: %d): %v", sts.Name, sts.Status.Replicas, *sts.Spec.Replicas, err)
	}
}

func updatePod(t *testing.T, podClient typedv1.PodInterface, podName string, updateFunc func(*v1.Pod)) *v1.Pod {
	var pod *v1.Pod
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		newPod, err := podClient.Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		updateFunc(newPod)
		pod, err = podClient.Update(context.TODO(), newPod, metav1.UpdateOptions{})
		return err
	}); err != nil {
		t.Fatalf("failed to update pod %s: %v", podName, err)
	}
	return pod
}

func updatePodStatus(t *testing.T, podClient typedv1.PodInterface, podName string, updateStatusFunc func(*v1.Pod)) *v1.Pod {
	var pod *v1.Pod
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		newPod, err := podClient.Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		updateStatusFunc(newPod)
		pod, err = podClient.UpdateStatus(context.TODO(), newPod, metav1.UpdateOptions{})
		return err
	}); err != nil {
		t.Fatalf("failed to update status of pod %s: %v", podName, err)
	}
	return pod
}

func getPods(t *testing.T, podClient typedv1.PodInterface, labelMap map[string]string) *v1.PodList {
	podSelector := labels.Set(labelMap).AsSelector()
	options := metav1.ListOptions{LabelSelector: podSelector.String()}
	pods, err := podClient.List(context.TODO(), options)
	if err != nil {
		t.Fatalf("failed obtaining a list of pods that match the pod labels %v: %v", labelMap, err)
	}
	if pods == nil {
		t.Fatalf("obtained a nil list of pods")
	}
	return pods
}

func updateSTS(t *testing.T, stsClient typedappsv1.StatefulSetInterface, stsName string, updateFunc func(*appsv1.StatefulSet)) *appsv1.StatefulSet {
	var sts *appsv1.StatefulSet
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		newSTS, err := stsClient.Get(context.TODO(), stsName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		updateFunc(newSTS)
		sts, err = stsClient.Update(context.TODO(), newSTS, metav1.UpdateOptions{})
		return err
	}); err != nil {
		t.Fatalf("failed to update sts %s: %v", stsName, err)
	}
	return sts
}

// Update .Spec.Replicas to replicas and verify .Status.Replicas is changed accordingly
func scaleSTS(t *testing.T, c pcclientset.Interface, sts *appsv1.StatefulSet, replicas int32) {
	stsClient := c.AppsV1().StatefulSets(sts.Namespace)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		newSTS, err := stsClient.Get(context.TODO(), sts.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*newSTS.Spec.Replicas = replicas
		sts, err = stsClient.Update(context.TODO(), newSTS, metav1.UpdateOptions{})
		return err
	}); err != nil {
		t.Fatalf("failed to update .Spec.Replicas to %d for sts %s: %v", replicas, sts.Name, err)
	}
	waitSTSStable(t, c, sts)
}

func scaleSTSWithDeleteSlots(t *testing.T, c pcclientset.Interface, sts *appsv1.StatefulSet, replicas int32, deleteSlots sets.Int32) {
	stsClient := c.AppsV1().StatefulSets(sts.Namespace)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		newSTS, err := stsClient.Get(context.TODO(), sts.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		helper.SetDeleteSlots(newSTS, deleteSlots)
		*newSTS.Spec.Replicas = replicas
		sts, err = stsClient.Update(context.TODO(), newSTS, metav1.UpdateOptions{})
		return err
	}); err != nil {
		t.Fatalf("failed to update .Spec.Replicas to %d for sts %s: %v", replicas, sts.Name, err)
	}
	waitSTSStable(t, c, sts)
}

func scaleInSTSByDeletingSlots(t *testing.T, c pcclientset.Interface, sts *appsv1.StatefulSet, ids ...int32) {
	set := sets.NewInt32(ids...)
	if set.Len() <= 0 {
		t.Fatalf("no slots")
	}
	stsClient := c.AppsV1().StatefulSets(sts.Namespace)
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		newSTS, err := stsClient.Get(context.TODO(), sts.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		helper.AddDeleteSlots(newSTS, set)
		replicas := *newSTS.Spec.Replicas - int32(set.Len())
		if replicas < 0 {
			t.Fatalf("replicas should not be less than 0: %v", replicas)
		}
		*newSTS.Spec.Replicas = replicas
		sts, err = stsClient.Update(context.TODO(), newSTS, metav1.UpdateOptions{})
		return err
	}); err != nil {
		t.Fatalf("failed to mark %d deleted for sts %s: %v", set, sts.Name, err)
	}
	waitSTSStable(t, c, sts)
}

func checkPodIdentifiers(t *testing.T, c clientset.Interface, sts *appsv1.StatefulSet, ids ...int32) {
	expectedIDs := sets.NewInt32(ids...)
	podClient := c.CoreV1().Pods(sts.Namespace)
	pods := getPods(t, podClient, labelMap())
	if len(pods.Items) != expectedIDs.Len() {
		t.Fatalf("len(pods) = %d, want %d", len(pods.Items), expectedIDs.Len())
	}
	gotIDs := sets.NewInt32()
	for _, pod := range pods.Items {
		klog.Infof("pod name %s\n", pod.Name)
		idStr := strings.TrimPrefix(pod.Name, sts.Name+"-")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			t.Fatalf("failed to parse string %q as an int: %v", idStr, err)
		}
		gotIDs.Insert(int32(id))
	}
	if !expectedIDs.Equal(gotIDs) {
		t.Fatalf("expected ids %v, got %v", expectedIDs.List(), gotIDs.List())
	}
}
