/*
Copyright 2022-2025 Kurator Authors.

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

package clusteroperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsKubeadmUpgradeSupported(t *testing.T) {
	testCases := []struct {
		name          string
		originVersion string
		targetVersion string
		expected      bool
	}{
		{
			name:          "Same major and minor version",
			originVersion: "v1.18.0",
			targetVersion: "v1.18.1",
			expected:      true,
		},
		{
			name:          "Skip one minor version",
			originVersion: "v1.18.0",
			targetVersion: "v1.19.0",
			expected:      true,
		},
		{
			name:          "Skip two minor versions",
			originVersion: "v1.18.0",
			targetVersion: "v1.20.0",
			expected:      false,
		},
		{
			name:          "Invalid origin version",
			originVersion: "invalid",
			targetVersion: "v1.18.1",
			expected:      false,
		},
		{
			name:          "Invalid target version",
			originVersion: "v1.18.0",
			targetVersion: "invalid",
			expected:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := isKubeadmUpgradeSupported(tc.originVersion, tc.targetVersion)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsSupportedVersion(t *testing.T) {
	testCases := []struct {
		name           string
		desiredVersion string
		minVersion     string
		maxVersion     string
		expected       bool
	}{
		{
			name:           "valid version within range",
			desiredVersion: "v1.2.3",
			minVersion:     "v1.0.0",
			maxVersion:     "v2.0.0",
			expected:       true,
		},
		{
			name:           "invalid desired version",
			desiredVersion: "invalid",
			minVersion:     "v1.0.0",
			maxVersion:     "v2.0.0",
			expected:       false,
		},
		{
			name:           "invalid min version",
			desiredVersion: "v1.2.3",
			minVersion:     "invalid",
			maxVersion:     "v2.0.0",
			expected:       false,
		},
		{
			name:           "invalid max version",
			desiredVersion: "v1.2.3",
			minVersion:     "v1.0.0",
			maxVersion:     "invalid",
			expected:       false,
		},
		{
			name:           "desired version below range",
			desiredVersion: "v0.9.9",
			minVersion:     "v1.0.0",
			maxVersion:     "v2.0.0",
			expected:       false,
		},
		{
			name:           "desired version above range",
			desiredVersion: "v2.1.0",
			minVersion:     "v1.0.0",
			maxVersion:     "v2.0.0",
			expected:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isSupportedVersion(tc.desiredVersion, tc.minVersion, tc.maxVersion)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGenerateUpgradeManageCMD(t *testing.T) {
	tests := []struct {
		name        string
		kubeVersion string
		expected    customClusterManageCMD
	}{
		{
			name:        "valid kube version",
			kubeVersion: "v1.20.0",
			expected:    "ansible-playbook -i inventory/cluster-hosts --private-key /root/.ssh/ssh-privatekey upgrade-cluster.yml -vvv  -e kube_version=v1.20.0",
		},
		{
			name:        "non-prefixed kube version",
			kubeVersion: "1.23.4",
			expected:    "ansible-playbook -i inventory/cluster-hosts --private-key /root/.ssh/ssh-privatekey upgrade-cluster.yml -vvv  -e kube_version=v1.23.4",
		},
		{
			name:        "empty kube version",
			kubeVersion: "",
			expected:    "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := generateUpgradeManageCMD(tc.kubeVersion)
			assert.Equal(t, tc.expected, got)
		})
	}
}
