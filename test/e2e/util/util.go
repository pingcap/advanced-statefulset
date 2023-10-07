// Copyright 2020 PingCAP, Inc.
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

package util

import (
	"fmt"

	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
)

var (
	ImageHttpd    = "httpd"
	ImageHttpdNew = "httpd-new"
	Images        = map[string]string{
		ImageHttpd:    "k8s.gcr.io/e2e-test-images/httpd:2.4.38-2",
		ImageHttpdNew: "k8s.gcr.io/e2e-test-images/httpd:2.4.39-2",
	}
)

// ServerVersionGTE returns true if v is greater than or equal to the server version.
func ServerVersionGTE(v *utilversion.Version, c discovery.ServerVersionInterface) (bool, error) {
	serverVersion, err := c.ServerVersion()
	if err != nil {
		return false, fmt.Errorf("Unable to get server version: %v", err)
	}
	sv, err := utilversion.ParseSemantic(serverVersion.GitVersion)
	if err != nil {
		return false, fmt.Errorf("Unable to parse server version %q: %v", serverVersion.GitVersion, err)
	}
	return sv.AtLeast(v), nil
}
