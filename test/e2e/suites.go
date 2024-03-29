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

/*
Copyright 2021 The Kubernetes Authors.

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

package e2e

import (
	"github.com/pingcap/advanced-statefulset/test/third_party/k8s"
)

// CleanupSuite is the boilerplate that can be used after tests on ginkgo were run, on the SynchronizedAfterSuite step.
// Similar to SynchronizedBeforeSuite, we want to run some operations only once (such as collecting cluster logs).
// Here, the order of functions is reversed; first, the function which runs everywhere,
// and then the function that only runs on the first Ginkgo node.
func CleanupSuite() {
	// Run on all Ginkgo nodes
	k8s.Logf("Running AfterSuite actions on all nodes")
	k8s.RunCleanupActions()
}

// AfterSuiteActions are actions that are run on ginkgo's SynchronizedAfterSuite
func AfterSuiteActions() {
	// Run only Ginkgo on node 1
	k8s.Logf("Running AfterSuite actions on node 1")
	if k8s.TestContext.ReportDir != "" {
		k8s.CoreDump(k8s.TestContext.ReportDir)
	}
	if k8s.TestContext.NodeKiller.Enabled {
		close(k8s.TestContext.NodeKiller.NodeKillerStopCh)
	}
}
