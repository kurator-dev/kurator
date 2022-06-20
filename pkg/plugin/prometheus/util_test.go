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

package prometheus

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenAdditionalScrapeConfigs(t *testing.T) {
	endpoints := []endpoint{
		{"remote1", "remote1.cluster"},
		{"remote2", "remote2.cluster"},
	}

	got, err := genAdditionalScrapeConfigs(endpoints)
	assert.NoError(t, err)

	expected, err := os.ReadFile(path.Join("testdata", "prometheus-additional.yaml"))
	assert.NoError(t, err)

	assert.Equal(t, string(expected), got)
}
