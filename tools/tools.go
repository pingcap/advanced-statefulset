// +build tools

// Tool dependencies are tracked here to make go module happy
// Refer https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
package tools

import (
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "k8s.io/code-generator"
)
