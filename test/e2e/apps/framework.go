/*
Copyright 2017 The Kubernetes Authors.

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

package apps

import "github.com/onsi/ginkgo"

// SIGDescribe annotates the test with the SIG label.
func SIGDescribe(text string, body func()) bool {
	return ginkgo.Describe("[sig-apps] "+text, body)
}

// KubeDescribe annotates the test with the Kube label.
// copied from https://github.com/kubernetes/kubernetes/blob/v1.20.15/test/e2e/framework/framework.go#L621-L625
func KubeDescribe(text string, body func()) bool {
	return ginkgo.Describe("[k8s.io] "+text, body)
}
