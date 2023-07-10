module github.com/pingcap/advanced-statefulset/client

go 1.19

require (
	github.com/google/go-cmp v0.5.2
	k8s.io/api v0.20.15
	k8s.io/apimachinery v0.20.15
	k8s.io/client-go v0.20.15
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.20.15
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible // indirect
	github.com/go-logr/logr v0.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.4.1 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/json-iterator/go v1.1.10 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/sys v0.0.0-20201112073958-5cba982894dd // indirect
	golang.org/x/text v0.3.6 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
	k8s.io/apiserver v0.20.15 // indirect
	k8s.io/component-base v0.20.15 // indirect
	k8s.io/klog/v2 v2.4.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211110013926-83f114cd0513 // indirect
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace k8s.io/api => k8s.io/api v0.20.15

replace k8s.io/apimachinery => k8s.io/apimachinery v0.20.15

replace k8s.io/client-go => k8s.io/client-go v0.20.15

replace k8s.io/code-generator => k8s.io/code-generator v0.20.15

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.15

replace k8s.io/apiserver => k8s.io/apiserver v0.20.15

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.15

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.20.15

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.20.15

replace k8s.io/component-base => k8s.io/component-base v0.20.15

replace k8s.io/cri-api => k8s.io/cri-api v0.20.15

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.20.15

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.20.15

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.20.15

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.20.15

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.20.15

replace k8s.io/kubectl => k8s.io/kubectl v0.20.15

replace k8s.io/kubelet => k8s.io/kubelet v0.20.15

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.20.15

replace k8s.io/metrics => k8s.io/metrics v0.20.15

replace k8s.io/node-api => k8s.io/node-api v0.20.15

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.20.15

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.20.15

replace k8s.io/sample-controller => k8s.io/sample-controller v0.20.15

replace k8s.io/mount-utils => k8s.io/mount-utils v0.20.15

replace k8s.io/component-helpers => k8s.io/component-helpers v0.20.15

replace k8s.io/controller-manager => k8s.io/controller-manager v0.20.15
