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

package application

import (
	"fmt"
	"reflect"
	"testing"

	flaggerv1b1 "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
	"github.com/fluxcd/flagger/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/fluxcd/flagger/pkg/apis/istio/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	applicationapi "kurator.dev/kurator/pkg/apis/apps/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/manifests"
)

func generateRolloutPloicy(installPrivateTestloader *bool) applicationapi.RolloutConfig {
	timeout := 50
	min := 99.0
	max := 500.0

	rolloutPolicy := applicationapi.RolloutConfig{
		TestLoader:             installPrivateTestloader,
		TrafficRoutingProvider: "istio",
		Workload: &applicationapi.CrossNamespaceObjectReference{
			APIVersion: "appv1/deployment",
			Kind:       "Deployment",
			Name:       "podinfo",
			Namespace:  "test",
		},
		ServiceName: "podinfo-service",
		Port:        80,
		RolloutPolicy: &applicationapi.RolloutPolicy{
			TrafficRouting: &applicationapi.TrafficRoutingConfig{
				TimeoutSeconds: 50,
				Gateways: []string{
					"istio-system/public-gateway",
				},
				Hosts: []string{
					"app.example.com",
				},
				Retries: &istiov1alpha3.HTTPRetry{
					Attempts:      10,
					PerTryTimeout: "40s",
					RetryOn:       "gateway-error, connect-failure, refused-stream",
				},
				Headers: &istiov1alpha3.Headers{
					Request: &istiov1alpha3.HeaderOperations{
						Add: map[string]string{
							"x-some-header": "value",
						},
					},
				},
				CorsPolicy: &istiov1alpha3.CorsPolicy{
					AllowOrigin:      []string{"example"},
					AllowMethods:     []string{"GET"},
					AllowCredentials: false,
					AllowHeaders:     []string{"x-some-header"},
					MaxAge:           "24h",
				},
				CanaryStrategy: &applicationapi.CanaryConfig{
					MaxWeight:  50,
					StepWeight: 10,
					StepWeights: []int{
						1, 20, 40, 80,
					},
					StepWeightPromotion: 30,
				},
				AnalysisTimes: 5,
				Match: []istiov1alpha3.HTTPMatchRequest{
					{
						Headers: map[string]v1alpha1.StringMatch{
							"user-agent": {
								Regex: ".*Firefox.*",
							},
							"cookie": {
								Regex: "^(.*?;)?(type=insider)(;.*)?$",
							},
						},
					},
				},
			},
			TrafficAnalysis: &applicationapi.TrafficAnalysis{
				CheckIntervalSeconds: &timeout,
				CheckFailedTimes:     &timeout,
				Metrics: []applicationapi.Metric{
					{
						Name:            "request-success-rate",
						IntervalSeconds: &timeout,
						ThresholdRange: &applicationapi.CanaryThresholdRange{
							Min: &min,
						},
					},
					{
						Name:            "request-duration",
						IntervalSeconds: &timeout,
						ThresholdRange: &applicationapi.CanaryThresholdRange{
							Max: &max,
						},
					},
				},
				Webhooks: applicationapi.Webhook{
					TimeoutSeconds: &timeout,
					Commands: []string{
						"hey -z 1m -q 10 -c 2 http://podinfo-canary.test:9898/",
						"curl -sd 'test' http://podinfo-canary:9898/token | grep token",
					},
				},
				SessionAffinity: &applicationapi.SessionAffinity{
					CookieName: "User",
					MaxAge:     24,
				},
			},
			RolloutTimeoutSeconds: &timeout,
			SkipTrafficAnalysis:   false,
			RevertOnDeletion:      false,
			Suspend:               false,
		},
	}
	return rolloutPolicy
}

func Test_renderCanary(t *testing.T) {
	int32Time := int32(50)
	sign := true
	type args struct {
		rolloutPolicy applicationapi.RolloutConfig
	}
	tests := []struct {
		name string
		args args
		want *flaggerv1b1.Canary
	}{
		{
			name: "functional test",
			args: args{
				rolloutPolicy: generateRolloutPloicy(&sign),
			},
			want: &flaggerv1b1.Canary{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "podinfo",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Canary",
					APIVersion: "flagger.app/v1beta1",
				},
				Spec: flaggerv1b1.CanarySpec{
					Provider: "istio",
					TargetRef: flaggerv1b1.LocalObjectReference{
						APIVersion: "appv1/deployment",
						Kind:       "Deployment",
						Name:       "podinfo",
					},
					ProgressDeadlineSeconds: &int32Time,
					SkipAnalysis:            false,
					RevertOnDeletion:        false,
					Suspend:                 false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderCanary(tt.args.rolloutPolicy, &flaggerv1b1.Canary{}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderCanary() = %v\n, want %v", got, tt.want)
			}
		})
	}
}

func Test_renderCanaryService(t *testing.T) {
	sign := true
	rolloutPolicy := generateRolloutPloicy(&sign)
	type args struct {
		rolloutPolicy applicationapi.RolloutConfig
		service       *corev1.Service
	}
	tests := []struct {
		name string
		args args
		want *flaggerv1b1.CanaryService
	}{
		{
			name: "functional test",
			args: args{
				rolloutPolicy: rolloutPolicy,
				service: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test",
						Name:      "podinfo-service",
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							"app": "podinfo",
						},
						Ports: []corev1.ServicePort{
							{
								Protocol:   corev1.ProtocolTCP,
								Port:       80,
								TargetPort: intstr.FromInt(8080),
							},
						},
					},
				},
			},
			want: &flaggerv1b1.CanaryService{
				Name:       "podinfo-service",
				Port:       80,
				Timeout:    "50s",
				TargetPort: intstr.FromInt(8080),
				Gateways: []string{
					"istio-system/public-gateway",
				},
				Hosts: []string{
					"app.example.com",
				},
				Retries:    rolloutPolicy.RolloutPolicy.TrafficRouting.Retries,
				Headers:    rolloutPolicy.RolloutPolicy.TrafficRouting.Headers,
				CorsPolicy: rolloutPolicy.RolloutPolicy.TrafficRouting.CorsPolicy,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := renderCanaryService(tt.args.rolloutPolicy, tt.args.service); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderCanaryService() = %v\n, want %v", got, tt.want)
			}
		})
	}
}

func Test_renderCanaryAnalysis(t *testing.T) {
	sign := true
	wantFalse := false
	timeout := 50
	rolloutPolicy := generateRolloutPloicy(&sign)
	wantPublicTestloaderRolloutPolicy := generateRolloutPloicy(&wantFalse)
	type args struct {
		rolloutPolicy applicationapi.RolloutConfig
	}
	tests := []struct {
		name string
		args args
		want *flaggerv1b1.CanaryAnalysis
	}{
		{
			name: "functional test",
			args: args{
				rolloutPolicy: rolloutPolicy,
			},
			want: &flaggerv1b1.CanaryAnalysis{
				Interval:   "50s",
				Iterations: 5,
				MaxWeight:  50,
				StepWeight: 10,
				StepWeights: []int{
					1, 20, 40, 80,
				},
				StepWeightPromotion: 30,
				Threshold:           timeout,
				Match: []istiov1alpha3.HTTPMatchRequest{
					{
						Headers: map[string]v1alpha1.StringMatch{
							"user-agent": {
								Regex: ".*Firefox.*",
							},
							"cookie": {
								Regex: "^(.*?;)?(type=insider)(;.*)?$",
							},
						},
					},
				},
				SessionAffinity: (*flaggerv1b1.SessionAffinity)(rolloutPolicy.RolloutPolicy.TrafficAnalysis.SessionAffinity),
				Metrics: []flaggerv1b1.CanaryMetric{
					{
						Name:           "request-success-rate",
						Interval:       "50s",
						ThresholdRange: (*flaggerv1b1.CanaryThresholdRange)(rolloutPolicy.RolloutPolicy.TrafficAnalysis.Metrics[0].ThresholdRange),
					},
					{
						Name:           "request-duration",
						Interval:       "50s",
						ThresholdRange: (*flaggerv1b1.CanaryThresholdRange)(rolloutPolicy.RolloutPolicy.TrafficAnalysis.Metrics[1].ThresholdRange),
					},
				},
				Webhooks: []flaggerv1b1.CanaryWebhook{
					{
						Name:    "generated-testload-0",
						Timeout: "50s",
						URL:     "http://podinfo-service-testloader.test/",
						Metadata: &map[string]string{
							"type": "cmd",
							"cmd":  "hey -z 1m -q 10 -c 2 http://podinfo-canary.test:9898/",
						},
					},
					{
						Name:    "generated-testload-1",
						Timeout: "50s",
						URL:     "http://podinfo-service-testloader.test/",
						Metadata: &map[string]string{
							"type": "cmd",
							"cmd":  "curl -sd 'test' http://podinfo-canary:9898/token | grep token",
						},
					},
				},
			},
		},
		{
			name: "public Testloader",
			args: args{
				rolloutPolicy: wantPublicTestloaderRolloutPolicy,
			},
			want: &flaggerv1b1.CanaryAnalysis{
				Interval:   "50s",
				Iterations: 5,
				MaxWeight:  50,
				StepWeight: 10,
				StepWeights: []int{
					1, 20, 40, 80,
				},
				StepWeightPromotion: 30,
				Threshold:           timeout,
				Match: []istiov1alpha3.HTTPMatchRequest{
					{
						Headers: map[string]v1alpha1.StringMatch{
							"user-agent": {
								Regex: ".*Firefox.*",
							},
							"cookie": {
								Regex: "^(.*?;)?(type=insider)(;.*)?$",
							},
						},
					},
				},
				SessionAffinity: (*flaggerv1b1.SessionAffinity)(rolloutPolicy.RolloutPolicy.TrafficAnalysis.SessionAffinity),
				Metrics: []flaggerv1b1.CanaryMetric{
					{
						Name:           "request-success-rate",
						Interval:       "50s",
						ThresholdRange: (*flaggerv1b1.CanaryThresholdRange)(rolloutPolicy.RolloutPolicy.TrafficAnalysis.Metrics[0].ThresholdRange),
					},
					{
						Name:           "request-duration",
						Interval:       "50s",
						ThresholdRange: (*flaggerv1b1.CanaryThresholdRange)(rolloutPolicy.RolloutPolicy.TrafficAnalysis.Metrics[1].ThresholdRange),
					},
				},
				Webhooks: []flaggerv1b1.CanaryWebhook{
					{
						Name:    "generated-testload-0",
						Timeout: "50s",
						URL:     "http://istio-system-testloader-kurator-member-loadtester.istio-system/",
						Metadata: &map[string]string{
							"type": "cmd",
							"cmd":  "hey -z 1m -q 10 -c 2 http://podinfo-canary.test:9898/",
						},
					},
					{
						Name:    "generated-testload-1",
						Timeout: "50s",
						URL:     "http://istio-system-testloader-kurator-member-loadtester.istio-system/",
						Metadata: &map[string]string{
							"type": "cmd",
							"cmd":  "curl -sd 'test' http://podinfo-canary:9898/token | grep token",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderCanaryAnalysis(tt.args.rolloutPolicy, "kurator-member"); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderCanaryAnalysis() = %v\n, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateDeployConfig(t *testing.T) {
	filepath := manifests.BuiltinOrDir("")
	//fmt.Printf("%s", filepath)
	deployname := "plugins/testloader-deploy.yaml"
	namespacedName := types.NamespacedName{
		Namespace: "test",
		Name:      "podinfo",
	}
	if _, err := generateDeployConfig(filepath, deployname, namespacedName.Name, namespacedName.Namespace); err != nil {
		fmt.Printf("failed get testloader deployment configuration: %v", err)
	}
}

func Test_generateSvcConfig(t *testing.T) {
	filepath := manifests.BuiltinOrDir("")
	//fmt.Printf("%s", filepath)
	svcname := "plugins/testloader-svc.yaml"
	namespacedName := types.NamespacedName{
		Namespace: "test",
		Name:      "podinfo",
	}
	if _, err := generateSvcConfig(filepath, svcname, namespacedName.Name, namespacedName.Namespace); err != nil {
		fmt.Printf("failed get testloader deployment configuration: %v", err)
	}
}
