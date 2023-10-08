/*
Copyright 2019 The Kubernetes Authors.

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

// this file is copied from k8s.io/kubernetes/test/e2e/framework/node/wait.go @v1.23.17

package node

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/pingcap/advanced-statefulset/test/third_party/k8s/log"
)

// waitListSchedulableNodes is a wrapper around listing nodes supporting retries.
func waitListSchedulableNodes(c clientset.Interface) (*v1.NodeList, error) {
	var nodes *v1.NodeList
	var err error
	if wait.PollImmediate(poll, singleCallTimeout, func() (bool, error) {
		nodes, err = c.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{FieldSelector: fields.Set{
			"spec.unschedulable": "false",
		}.AsSelector().String()})
		if err != nil {
			return false, err
		}
		return true, nil
	}) != nil {
		return nodes, err
	}
	return nodes, nil
}

// checkWaitListSchedulableNodes is a wrapper around listing nodes supporting retries.
func checkWaitListSchedulableNodes(c clientset.Interface) (*v1.NodeList, error) {
	nodes, err := waitListSchedulableNodes(c)
	if err != nil {
		return nil, fmt.Errorf("error: %s. Non-retryable failure or timed out while listing nodes for e2e cluster", err)
	}
	return nodes, nil
}

// CheckReadyForTests returns a function which will return 'true' once the number of ready nodes is above the allowedNotReadyNodes threshold (i.e. to be used as a global gate for starting the tests).
func CheckReadyForTests(c clientset.Interface, nonblockingTaints string, allowedNotReadyNodes, largeClusterThreshold int) func() (bool, error) {
	attempt := 0
	return func() (bool, error) {
		if allowedNotReadyNodes == -1 {
			return true, nil
		}
		attempt++
		var nodesNotReadyYet []v1.Node
		opts := metav1.ListOptions{
			ResourceVersion: "0",
			// remove uncordoned nodes from our calculation, TODO refactor if node v2 API removes that semantic.
			FieldSelector: fields.Set{"spec.unschedulable": "false"}.AsSelector().String(),
		}
		allNodes, err := c.CoreV1().Nodes().List(context.TODO(), opts)
		if err != nil {
			var terminalListNodesErr error
			log.Logf("Unexpected error listing nodes: %v", err)
			if attempt >= 3 {
				terminalListNodesErr = err
			}
			return false, terminalListNodesErr
		}
		for _, node := range allNodes.Items {
			if !readyForTests(&node, nonblockingTaints) {
				nodesNotReadyYet = append(nodesNotReadyYet, node)
			}
		}
		// Framework allows for <TestContext.AllowedNotReadyNodes> nodes to be non-ready,
		// to make it possible e.g. for incorrect deployment of some small percentage
		// of nodes (which we allow in cluster validation). Some nodes that are not
		// provisioned correctly at startup will never become ready (e.g. when something
		// won't install correctly), so we can't expect them to be ready at any point.
		//
		// We log the *reason* why nodes are not schedulable, specifically, its usually the network not being available.
		if len(nodesNotReadyYet) > 0 {
			// In large clusters, log them only every 10th pass.
			if len(nodesNotReadyYet) < largeClusterThreshold || attempt%10 == 0 {
				log.Logf("Unschedulable nodes= %v, maximum value for starting tests= %v", len(nodesNotReadyYet), allowedNotReadyNodes)
				for _, node := range nodesNotReadyYet {
					log.Logf("	-> Node %s [[[ Ready=%t, Network(available)=%t, Taints=%v, NonblockingTaints=%v ]]]",
						node.Name,
						IsConditionSetAsExpectedSilent(&node, v1.NodeReady, true),
						IsConditionSetAsExpectedSilent(&node, v1.NodeNetworkUnavailable, false),
						node.Spec.Taints,
						nonblockingTaints,
					)

				}
				if len(nodesNotReadyYet) > allowedNotReadyNodes {
					ready := len(allNodes.Items) - len(nodesNotReadyYet)
					remaining := len(nodesNotReadyYet) - allowedNotReadyNodes
					log.Logf("==== node wait: %v out of %v nodes are ready, max notReady allowed %v.  Need %v more before starting.", ready, len(allNodes.Items), allowedNotReadyNodes, remaining)
				}
			}
		}
		return len(nodesNotReadyYet) <= allowedNotReadyNodes, nil
	}
}

// readyForTests determines whether or not we should continue waiting for the nodes
// to enter a testable state. By default this means it is schedulable, NodeReady, and untainted.
// Nodes with taints nonblocking taints are permitted to have that taint and
// also have their node.Spec.Unschedulable field ignored for the purposes of this function.
func readyForTests(node *v1.Node, nonblockingTaints string) bool {
	if hasNonblockingTaint(node, nonblockingTaints) {
		// If the node has one of the nonblockingTaints taints; just check that it is ready
		// and don't require node.Spec.Unschedulable to be set either way.
		if !IsNodeReady(node) || !isNodeUntaintedWithNonblocking(node, nonblockingTaints) {
			return false
		}
	} else {
		if !IsNodeSchedulable(node) || !isNodeUntainted(node) {
			return false
		}
	}
	return true
}
