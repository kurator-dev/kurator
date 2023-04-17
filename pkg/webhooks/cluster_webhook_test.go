/*
Copyright Kurator Authors.

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

package webhooks

import (
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/yaml"

	v1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

func TestClusterValidation(t *testing.T) {
	// read configuration from examples directory
	r := path.Join("../../examples", "cluster")
	caseNames := make([]string, 0)
	err := filepath.WalkDir(r, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		caseNames = append(caseNames, path)

		return nil
	})
	assert.NoError(t, err)

	wh := &ClusterWebhook{}
	for _, tt := range caseNames {
		t.Run(tt, func(t *testing.T) {
			g := NewWithT(t)
			c, err := readCluster(tt)
			g.Expect(err).NotTo(HaveOccurred())

			err = wh.validate(c)
			g.Expect(err).NotTo(HaveOccurred())
		})
	}
}

func TestInvalidClusterValidation(t *testing.T) {
	r := path.Join("testdata", "cluster")
	caseNames := make([]string, 0)
	err := filepath.WalkDir(r, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		caseNames = append(caseNames, path)

		return nil
	})
	assert.NoError(t, err)

	wh := &ClusterWebhook{}
	for _, tt := range caseNames {
		t.Run(tt, func(t *testing.T) {
			g := NewWithT(t)
			c, err := readCluster(tt)
			g.Expect(err).NotTo(HaveOccurred())

			err = wh.validate(c)
			g.Expect(err).To(HaveOccurred())
			t.Logf("%v", err)
		})
	}
}

func TestUpdateClusterInfraType(t *testing.T) {
	wh := &ClusterWebhook{}
	oldCluster, err := readCluster("../../examples/cluster/quickstart.yaml")
	assert.NoError(t, err)

	newCluster := oldCluster.DeepCopy()
	newCluster.Spec.InfraType = "aws1"

	err = wh.ValidateUpdate(context.TODO(), oldCluster, newCluster)
	if !apierrors.IsInvalid(err) {
		t.Errorf("Expect an invalid error, got %v", err)
	}
}

func readCluster(filename string) (*v1.Cluster, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &v1.Cluster{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}

	return c, nil
}
