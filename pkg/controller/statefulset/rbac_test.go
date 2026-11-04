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

package statefulset

import (
	"io"
	"os"
	"testing"

	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestControllerRBACLeastPrivilege(t *testing.T) {
	f, err := os.Open("../../../manifests/rbac.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	decoder := yaml.NewYAMLOrJSONDecoder(f, 4096)
	var cluster, leader []rbac.PolicyRule
	for {
		var obj struct {
			Kind  string            `json:"kind"`
			Rules []rbac.PolicyRule `json:"rules"`
		}
		err := decoder.Decode(&obj)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch obj.Kind {
		case "ClusterRole":
			cluster = obj.Rules
		case "Role":
			leader = obj.Rules
		}
	}
	allows := func(rules []rbac.PolicyRule, group, resource, verb string) bool {
		for _, rule := range rules {
			match := func(values []string, want string) bool {
				for _, v := range values {
					if v == want || v == "*" {
						return true
					}
				}
				return false
			}
			if match(rule.APIGroups, group) && match(rule.Resources, resource) && match(rule.Verbs, verb) {
				return true
			}
		}
		return false
	}
	for _, rule := range append(append([]rbac.PolicyRule{}, cluster...), leader...) {
		for _, values := range [][]string{rule.APIGroups, rule.Resources, rule.Verbs} {
			for _, v := range values {
				if v == "*" {
					t.Fatal("wildcard permission")
				}
			}
		}
	}
	for _, tc := range []struct{ group, resource, verb string }{
		{"", "persistentvolumes", "get"}, {"", "secrets", "get"}, {"", "pods/exec", "create"}, {"", "pods", "deletecollection"},
		{"apps.pingcap.com", "statefulsets", "delete"}, {"apps.pingcap.com", "statefulsets", "create"},
	} {
		if allows(cluster, tc.group, tc.resource, tc.verb) {
			t.Errorf("excess permission: %+v", tc)
		}
	}
	if !allows(cluster, "apps.pingcap.com", "statefulsets/finalizers", "update") {
		t.Fatal("owner-reference admission permission missing")
	}
	if !allows(cluster, "apps.pingcap.com", "statefulsets/status", "update") {
		t.Fatal("status permission missing")
	}
	for _, verb := range []string{"get", "create", "update"} {
		if !allows(leader, "coordination.k8s.io", "leases", verb) {
			t.Fatal("lease permission missing", verb)
		}
	}
	if allows(leader, "", "endpoints", "get") {
		t.Fatal("unused endpoints permission retained")
	}
}
