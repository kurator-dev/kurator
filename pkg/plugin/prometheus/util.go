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
	"fmt"
	"net/url"

	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	promcfg "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"gopkg.in/yaml.v3"
)

type endpoint struct {
	name    string
	address string
}

func genAdditionalScrapeConfigs(endpoints []endpoint) (string, error) {
	federalScrapeConfigs := make([]*promcfg.ScrapeConfig, 0, len(endpoints))
	for _, ep := range endpoints {
		sc := &promcfg.ScrapeConfig{
			JobName:          ep.name,
			MetricsPath:      "/federate",
			HTTPClientConfig: config.DefaultHTTPClientConfig,
			Params: url.Values{
				"match[]": []string{
					`{__name__=~".+"}`,
				},
			},
		}
		sc.ServiceDiscoveryConfigs = discovery.Configs{
			discovery.StaticConfig{
				{
					Targets: []model.LabelSet{
						{model.AddressLabel: model.LabelValue(fmt.Sprintf("%s:9090", ep.address))},
					},
					Labels: model.LabelSet{
						"cluster": model.LabelValue(ep.name),
					},
				},
			},
		}

		federalScrapeConfigs = append(federalScrapeConfigs, sc)
	}

	cfg, err := yaml.Marshal(federalScrapeConfigs)
	if err != nil {
		return "", err
	}
	return string(cfg), nil
}
