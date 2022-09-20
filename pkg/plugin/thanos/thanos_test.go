package thanos

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestClusterOverridePolicy(t *testing.T) {
	op := clusterOverridePolicy("cluster1")

	b, err := yaml.Marshal(op)
	assert.NoError(t, err)

	expect, err := os.ReadFile("./testdata/external_labels_overridepolicy.yaml")
	assert.NoError(t, err)

	assert.Equal(t, string(expect), string(b))
}
