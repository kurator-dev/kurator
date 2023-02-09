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

package names

import (
	"fmt"
	"hash/fnv"

	hashutil "github.com/karmada-io/karmada/pkg/util/hash"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	apinames "k8s.io/apiserver/pkg/storage/names"
)

type Generator interface {
	Generate(nn types.NamespacedName) string
}

var _ Generator = &simpleGenerator{}

func NewSimpleGenerator() Generator {
	return &simpleGenerator{
		namer: apinames.SimpleNameGenerator,
	}
}

type simpleGenerator struct {
	namer apinames.NameGenerator
}

func (g *simpleGenerator) Generate(nn types.NamespacedName) string {
	hash := fnv.New32a()
	hashutil.DeepHashObject(hash, nn.String())
	return fmt.Sprintf("%s-%s-%s",
		nn.Namespace, nn.Name,
		rand.SafeEncodeString(fmt.Sprint(hash.Sum32())))
}
