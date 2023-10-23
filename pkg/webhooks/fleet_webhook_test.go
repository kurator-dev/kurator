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
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	"kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

func TestValidFleetValidation(t *testing.T) {
	r := path.Join("../../examples", "fleet")
	caseNames := getCaseNames(t, r)

	wh := &FleetWebhook{}
	for _, tt := range caseNames {
		t.Run(tt, func(t *testing.T) {
			g := NewWithT(t)
			c, err := readFleet(tt)
			g.Expect(err).NotTo(HaveOccurred())

			err = wh.validate(c)
			g.Expect(err).NotTo(HaveOccurred())
		})
	}
}

func TestInvalidFleetValidation(t *testing.T) {
	r := path.Join("testdata", "fleet")
	caseNames := getCaseNames(t, r)

	wh := &FleetWebhook{}
	for _, tt := range caseNames {
		t.Run(tt, func(t *testing.T) {
			g := NewWithT(t)
			c, err := readFleet(tt)
			g.Expect(err).NotTo(HaveOccurred())

			err = wh.validate(c)
			g.Expect(err).To(HaveOccurred())
			t.Logf("%v", err)
		})
	}
}

func getCaseNames(t *testing.T, rootPath string) []string {
	caseNames := make([]string, 0)
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		caseNames = append(caseNames, path)
		return nil
	})
	assert.NoError(t, err)
	return caseNames
}

func readFleet(filename string) (*v1alpha1.Fleet, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &v1alpha1.Fleet{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}

	return c, nil
}
