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

package application

import (
	"reflect"
	"testing"

	flaggerv1b1 "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
	"github.com/fluxcd/flagger/pkg/apis/istio/common/v1alpha1"
	istiov1alpha3 "github.com/fluxcd/flagger/pkg/apis/istio/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	applicationapi "kurator.dev/kurator/pkg/apis/apps/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

func generateRolloutPolicy(installPrivateTestloader *bool) applicationapi.RolloutConfig {
	timeout := 50
	RolloutTimeoutSeconds := int32(50)
	min := 99.0
	max := 500.0

	rolloutPolicy := applicationapi.RolloutConfig{
		TestLoader:             installPrivateTestloader,
		TrafficRoutingProvider: fleetapi.Istio,
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
			RolloutTimeoutSeconds: &RolloutTimeoutSeconds,
			SkipTrafficAnalysis:   false,
			RevertOnDeletion:      false,
			Suspend:               false,
		},
	}
	return rolloutPolicy
}

func generateRolloutPolicyWithCustomMetric() applicationapi.RolloutConfig {
	timeout := 50
	RolloutTimeoutSeconds := int32(50)
	min := 99.0
	max := 500.0
	flag := false

	rolloutPolicy := applicationapi.RolloutConfig{
		TestLoader:             &flag,
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
						Name:            "my-metric",
						IntervalSeconds: &timeout,
						ThresholdRange: &applicationapi.CanaryThresholdRange{
							Max: &max,
						},
						CustomMetric: &flaggerv1b1.MetricTemplateSpec{
							Provider: flaggerv1b1.MetricTemplateProvider{
								Type:    "prometheus",
								Address: "http://flagger-prometheus.ingress-nginx:9090",
							},
							Query: `
                			   sum(
                			     rate(
                			       http_requests_total{
                			         status!~"5.*"
                			       }[{{ interval }}]
                			     )
                			   )
                			   /
                			   sum(
                			     rate(
                			       http_requests_total[{{ interval }}]
                			     )
                			   ) * 100`,
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
			RolloutTimeoutSeconds: &RolloutTimeoutSeconds,
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
				rolloutPolicy: generateRolloutPolicy(&sign),
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
	rolloutPolicy := generateRolloutPolicy(&sign)
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
	rolloutPolicy := generateRolloutPolicy(&sign)
	wantPublicTestloaderRolloutPolicy := generateRolloutPolicy(&wantFalse)
	rolloutPolicyWithCustomMetric := generateRolloutPolicyWithCustomMetric()
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
		{
			name: "Custom Metric Template",
			args: args{
				rolloutPolicy: rolloutPolicyWithCustomMetric,
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
						Name:           "my-metric",
						Interval:       "50s",
						ThresholdRange: (*flaggerv1b1.CanaryThresholdRange)(rolloutPolicy.RolloutPolicy.TrafficAnalysis.Metrics[1].ThresholdRange),
						TemplateRef: &flaggerv1b1.CrossNamespaceObjectReference{
							Name:      "my-metric",
							Namespace: rolloutPolicyWithCustomMetric.Workload.Namespace,
						},
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

func Test_addLables(t *testing.T) {
	type args struct {
		obj   client.Object
		label map[string]string
	}
	tests := []struct {
		name string
		args args
		want client.Object
	}{
		{
			name: "function test",
			args: args{
				obj: &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Namespace",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "webapp",
						Labels: map[string]string{
							"xxx": "abc",
						},
					},
				},
				label: map[string]string{
					"istio-injection": "enabled",
				},
			},
			want: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "webapp",
					Labels: map[string]string{
						"xxx":             "abc",
						"istio-injection": "enabled",
					},
				},
			},
		},
		{
			name: "empty labels test",
			args: args{
				obj: &corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Namespace",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "webapp",
					},
				},
				label: map[string]string{"XXX": "abc"},
			},
			want: &corev1.Namespace{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "webapp",
					Labels: map[string]string{
						"XXX": "abc",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := addLabels(tt.args.obj, tt.args.label); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addLablesOrAnnotaions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeMap(t *testing.T) {
	type args struct {
		map1 map[string]*applicationapi.RolloutStatus
		map2 map[string]*applicationapi.RolloutStatus
	}
	tests := []struct {
		name string
		args args
		want map[string]*applicationapi.RolloutStatus
	}{
		{
			name: "function test",
			args: args{
				map1: map[string]*applicationapi.RolloutStatus{
					"kurator": {
						ClusterName:          "kurator-member1",
						RolloutNameInCluster: "podinfo",
						RolloutStatusInCluster: &flaggerv1b1.CanaryStatus{
							Phase: "success",
						},
					},
					"istio": {
						ClusterName:          "kurator-member2",
						RolloutNameInCluster: "podinfo",
						RolloutStatusInCluster: &flaggerv1b1.CanaryStatus{
							Phase: "Initializing",
						},
					},
				},
				map2: map[string]*applicationapi.RolloutStatus{
					"kubeedge": {
						ClusterName:          "kurator-member1",
						RolloutNameInCluster: "podinfo",
						RolloutStatusInCluster: &flaggerv1b1.CanaryStatus{
							Phase: "success",
						},
					},
					"karmada": {
						ClusterName:          "kurator-member1",
						RolloutNameInCluster: "podinfo",
						RolloutStatusInCluster: &flaggerv1b1.CanaryStatus{
							Phase: "Initializing",
						},
					},
				},
			},
			want: map[string]*applicationapi.RolloutStatus{
				"kurator": {
					ClusterName:          "kurator-member1",
					RolloutNameInCluster: "podinfo",
					RolloutStatusInCluster: &flaggerv1b1.CanaryStatus{
						Phase: "success",
					},
				},
				"istio": {
					ClusterName:          "kurator-member2",
					RolloutNameInCluster: "podinfo",
					RolloutStatusInCluster: &flaggerv1b1.CanaryStatus{
						Phase: "Initializing",
					},
				},
				"kubeedge": {
					ClusterName:          "kurator-member1",
					RolloutNameInCluster: "podinfo",
					RolloutStatusInCluster: &flaggerv1b1.CanaryStatus{
						Phase: "success",
					},
				},
				"karmada": {
					ClusterName:          "kurator-member1",
					RolloutNameInCluster: "podinfo",
					RolloutStatusInCluster: &flaggerv1b1.CanaryStatus{
						Phase: "Initializing",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeMap(tt.args.map1, tt.args.map2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
