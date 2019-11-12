package apps

import (
	"fmt"
	"reflect"
	"time"

	pcv1alpha1 "github.com/cofyc/advanced-statefulset/pkg/apis/apps/v1alpha1"
	"github.com/cofyc/advanced-statefulset/pkg/apis/apps/v1alpha1/helper"
	asclientset "github.com/cofyc/advanced-statefulset/pkg/client/clientset/versioned"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2eservice "k8s.io/kubernetes/test/e2e/framework/service"
	e2esset "k8s.io/kubernetes/test/e2e/framework/statefulset"
)

var _ = SIGDescribe("StatefulSet", func() {
	f := framework.NewDefaultFramework("statefulset")
	var ns string
	var c clientset.Interface
	var asc asclientset.Interface

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		c = f.ClientSet
		config, err := framework.LoadConfig()
		framework.ExpectNoError(err)
		asc, err = asclientset.NewForConfig(config)
		framework.ExpectNoError(err)
	})

	ginkgo.Describe("Basic Advanced StatefulSet functionality [AdvancedStatefulSetBasic]", func() {
		ssName := "ss"
		ssLabels := map[string]string{
			"foo": "bar",
			"baz": "blah",
		}
		headlessSvcName := "test"
		var statefulPodMounts, podMounts []v1.VolumeMount
		var ss *appsv1.StatefulSet

		ginkgo.BeforeEach(func() {
			statefulPodMounts = []v1.VolumeMount{{Name: "datadir", MountPath: "/data/"}}
			podMounts = []v1.VolumeMount{{Name: "home", MountPath: "/home"}}
			ss = e2esset.NewStatefulSet(ssName, ns, headlessSvcName, 2, statefulPodMounts, podMounts, ssLabels)

			ginkgo.By("Creating service " + headlessSvcName + " in namespace " + ns)
			headlessService := e2eservice.CreateServiceSpec(headlessSvcName, "", true, ssLabels)
			_, err := c.CoreV1().Services(ns).Create(headlessService)
			framework.ExpectNoError(err)
		})

		ginkgo.AfterEach(func() {
			if ginkgo.CurrentGinkgoTestDescription().Failed {
				framework.DumpDebugInfo(c, ns)
			}
			framework.Logf("Deleting all advanced statefulset in ns %v", ns)
			e2esset.DeleteAllStatefulSets(helper.NewHijackClient(f.ClientSet, asc), ns)
		})

		ginkgo.It("migrate from Kubernetes StatefulSet", func() {
			ginkgo.By("Creating statefulset " + ssName + " in namespace " + ns)
			*(ss.Spec.Replicas) = 3
			e2esset.PauseNewPods(ss)

			ss, err := c.AppsV1().StatefulSets(ns).Create(ss)
			framework.ExpectNoError(err)

			ginkgo.By("Saturating stateful set " + ss.Name)
			e2esset.Saturate(c, ss)

			ginkgo.By(fmt.Sprintf("Orphan and save controller revisions"))
			selector, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector)
			framework.ExpectNoError(err)
			revisionListOptions := metav1.ListOptions{LabelSelector: selector.String()}
			oldRevisionList, err := c.AppsV1().ControllerRevisions(ns).List(revisionListOptions)
			framework.ExpectNoError(err)
			migrateLabel := "apps.pingcap.com/migrate"
			for _, revision := range oldRevisionList.Items {
				// It's important to empty statefulset selector labels,
				// otherwise sts will adopt it again on delete event and then
				// GC will delete revisions because they are not orphans.
				// https://github.com/kubernetes/kubernetes/issues/84982
				for key := range ss.Spec.Selector.MatchLabels {
					delete(revision.Labels, key)
				}
				revision.Labels[migrateLabel] = ss.Name // a special label to mark it should be adopted by the new sts
				_, err = c.AppsV1().ControllerRevisions(ns).Update(&revision)
				framework.ExpectNoError(err)
			}

			ginkgo.By(fmt.Sprintf("Deleting stateful set %q without dependents", ss.Name))
			policy := metav1.DeletePropagationOrphan
			c.AppsV1().StatefulSets(ns).Delete(ss.Name, &metav1.DeleteOptions{
				PropagationPolicy: &policy,
			})

			migrateSelector := labels.SelectorFromSet(map[string]string{
				migrateLabel: ss.Name,
			})
			migrateRevisionListOption := metav1.ListOptions{LabelSelector: migrateSelector.String()}

			ginkgo.By("Waiting for pods/controllerrevisions to be orphaned")
			err = wait.PollImmediate(time.Second, time.Minute, func() (done bool, err error) {
				podList := e2esset.GetPodList(c, ss)
				gomega.Expect(podList.Items).To(gomega.HaveLen(int(*ss.Spec.Replicas)))
				for _, pod := range podList.Items {
					if metav1.GetControllerOf(&pod) != nil {
						return false, nil
					}
				}
				revisionList, err := c.AppsV1().ControllerRevisions(ns).List(migrateRevisionListOption)
				framework.ExpectNoError(err)
				gomega.Expect(revisionList.Items).To(gomega.HaveLen(len(oldRevisionList.Items)))

				for _, revision := range revisionList.Items {
					if metav1.GetControllerOf(&revision) != nil {
						return false, nil
					}
				}
				return true, nil
			})
			framework.ExpectNoError(err)

			ginkgo.By("Wait for a while that pods/controllerrevisions should not be deleted")
			err = wait.PollImmediate(time.Second, time.Minute, func() (done bool, err error) {
				podList := e2esset.GetPodList(c, ss)
				gomega.Expect(podList.Items).To(gomega.HaveLen(int(*ss.Spec.Replicas)))
				for _, pod := range podList.Items {
					gomega.Expect(pod.Status.Phase).To(gomega.Equal(v1.PodRunning))
					gomega.Expect(metav1.GetControllerOf(&pod)).To(gomega.BeNil())
				}

				revisionList, err := c.AppsV1().ControllerRevisions(ns).List(migrateRevisionListOption)
				framework.ExpectNoError(err)
				gomega.Expect(revisionList.Items).To(gomega.HaveLen(len(oldRevisionList.Items)))
				for _, revision := range revisionList.Items {
					gomega.Expect(metav1.GetControllerOf(&revision)).To(gomega.BeNil())
				}
				return false, nil
			})
			framework.ExpectEqual(err, wait.ErrWaitTimeout)

			ginkgo.By(fmt.Sprintf("Creating a new advanced sts %q", ss.Name))
			asts, err := helper.FromBuiltinStatefulSet(ss)
			framework.ExpectNoError(err)
			asts.ObjectMeta = metav1.ObjectMeta{
				Name:      ss.Name,
				Namespace: ss.Namespace,
			}
			asts.Status = pcv1alpha1.StatefulSetStatus{}
			asts, err = asc.AppsV1alpha1().StatefulSets(ns).Create(asts)
			framework.ExpectNoError(err)

			ginkgo.By("Recovering controller revisions")
			revisionList, err := c.AppsV1().ControllerRevisions(ns).List(migrateRevisionListOption)
			framework.ExpectNoError(err)
			for _, revision := range revisionList.Items {
				delete(revision.Labels, migrateLabel)
				for key, val := range asts.Spec.Selector.MatchLabels {
					revision.Labels[key] = val
				}
				_, err = c.AppsV1().ControllerRevisions(ns).Update(&revision)
				framework.ExpectNoError(err)
			}

			ginkgo.By(fmt.Sprintf("Wait for pods/controllerrevisions to be adopted"))
			controllerKind := pcv1alpha1.SchemeGroupVersion.WithKind("StatefulSet")
			controllerRef := metav1.NewControllerRef(asts, controllerKind)
			err = wait.PollImmediate(time.Second, time.Minute, func() (done bool, err error) {
				podList := e2esset.GetPodList(c, ss)
				gomega.Expect(podList.Items).To(gomega.HaveLen(int(*ss.Spec.Replicas)))
				for _, pod := range podList.Items {
					ref := metav1.GetControllerOf(&pod)
					if ref == nil {
						return false, nil
					}
					if !reflect.DeepEqual(ref, controllerRef) {
						return false, nil
					}
				}
				revisionList, err := c.AppsV1().ControllerRevisions(ns).List(revisionListOptions)
				framework.ExpectNoError(err)
				gomega.Expect(revisionList.Items).To(gomega.HaveLen(len(oldRevisionList.Items)))

				for _, revision := range revisionList.Items {
					ref := metav1.GetControllerOf(&revision)
					if ref == nil {
						return false, nil
					}
					if !reflect.DeepEqual(ref, controllerRef) {
						return false, nil
					}
				}
				return true, nil
			})
			framework.ExpectNoError(err)
		})
	})

})
