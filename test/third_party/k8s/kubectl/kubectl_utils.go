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

// this file is copied from k8s.io/kubernetes/test/e2e/framework/kubectl/kubectl_utils.go @v1.23.17

package kubectl

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
)

// TestKubeconfig is a struct containing the needed attributes from TestContext and Framework(Namespace).
type TestKubeconfig struct {
	CertDir     string
	Host        string
	KubeConfig  string
	KubeContext string
	KubectlPath string
	Namespace   string // Every test has at least one namespace unless creation is skipped
}

// NewTestKubeconfig returns a new Kubeconfig struct instance.
func NewTestKubeconfig(certdir, host, kubeconfig, kubecontext, kubectlpath, namespace string) *TestKubeconfig {
	return &TestKubeconfig{
		CertDir:     certdir,
		Host:        host,
		KubeConfig:  kubeconfig,
		KubeContext: kubecontext,
		KubectlPath: kubectlpath,
		Namespace:   namespace,
	}
}

// KubectlCmd runs the kubectl executable through the wrapper script.
func (tk *TestKubeconfig) KubectlCmd(args ...string) *exec.Cmd {
	defaultArgs := []string{}

	// Reference a --server option so tests can run anywhere.
	if tk.Host != "" {
		defaultArgs = append(defaultArgs, "--"+clientcmd.FlagAPIServer+"="+tk.Host)
	}
	if tk.KubeConfig != "" {
		defaultArgs = append(defaultArgs, "--"+clientcmd.RecommendedConfigPathFlag+"="+tk.KubeConfig)

		// Reference the KubeContext
		if tk.KubeContext != "" {
			defaultArgs = append(defaultArgs, "--"+clientcmd.FlagContext+"="+tk.KubeContext)
		}

	} else {
		if tk.CertDir != "" {
			defaultArgs = append(defaultArgs,
				fmt.Sprintf("--certificate-authority=%s", filepath.Join(tk.CertDir, "ca.crt")),
				fmt.Sprintf("--client-certificate=%s", filepath.Join(tk.CertDir, "kubecfg.crt")),
				fmt.Sprintf("--client-key=%s", filepath.Join(tk.CertDir, "kubecfg.key")))
		}
	}
	if tk.Namespace != "" {
		defaultArgs = append(defaultArgs, fmt.Sprintf("--namespace=%s", tk.Namespace))
	}
	kubectlArgs := append(defaultArgs, args...)

	//We allow users to specify path to kubectl, so you can test either "kubectl" or "cluster/kubectl.sh"
	//and so on.
	cmd := exec.Command(tk.KubectlPath, kubectlArgs...)

	//caller will invoke this and wait on it.
	return cmd
}
