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

package render

import (
	"bytes"
	"text/template"

	"k8s.io/apimachinery/pkg/types"
)

type TestloaderConfig struct {
	Name            string
	Namespace       string
	LabelKey        string
	LabelValue      string
	AnnotationKey   string
	AnnotationValue string
}

const testloaderTemplateName = "testloader"

// RenderTestloaderConfig takes generates YAML byte array configuration representing the testloader configuration.
func RenderTestloaderConfig(constTemplateName string, namespacedName types.NamespacedName, annotationKey, annotationsValue string) ([]byte, error) {
	cfg := TestloaderConfig{
		Name:            namespacedName.Name,
		Namespace:       namespacedName.Namespace,
		LabelKey:        "app",
		LabelValue:      namespacedName.Name,
		AnnotationKey:   annotationKey,
		AnnotationValue: annotationsValue,
	}

	return renderTestloaderTemplateConfig(constTemplateName, cfg)
}

// renderTestloaderTemplateConfig reads, parses, and renders a template file using the provided configuration data.
func renderTestloaderTemplateConfig(constTemplateName string, cfg TestloaderConfig) ([]byte, error) {
	tql, err := template.New(testloaderTemplateName).Funcs(funMap()).Parse(constTemplateName)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := tql.Execute(&b, cfg); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func funMap() template.FuncMap {
	return nil
}

const TestlaoderDeployment = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
  labels:
    {{.LabelKey}}: {{.LabelValue}}
  annotations:
    {{.AnnotationKey}}: {{.AnnotationValue}}
spec:
  selector:
    matchLabels:
      {{.LabelKey}}: {{.LabelValue}}
  template:
    metadata:
      labels:
        {{.LabelKey}}: {{.LabelValue}}
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        openservicemesh.io/inbound-port-exclusion-list: "80, 8080"
    spec:
      containers:
        - name: loadtester
          image: ghcr.io/fluxcd/flagger-loadtester:0.29.0
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 8080
          command:
            - ./loadtester
            - -port=8080
            - -log-level=info
            - -timeout=1h
          livenessProbe:
            exec:
              command:
                - wget
                - --quiet
                - --tries=1
                - --timeout=4
                - --spider
                - http://localhost:8080/healthz
            timeoutSeconds: 5
          readinessProbe:
            exec:
              command:
                - wget
                - --quiet
                - --tries=1
                - --timeout=4
                - --spider
                - http://localhost:8080/healthz
            timeoutSeconds: 5
          resources:
            limits:
              memory: "512Mi"
              cpu: "1000m"
            requests:
              memory: "32Mi"
              cpu: "10m"
          securityContext:
            readOnlyRootFilesystem: true
            runAsUser: 10001
`

const TestlaoderService = `apiVersion: v1
kind: Service
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
  labels:
    {{.LabelKey}}: {{.LabelValue}}
  annotations:
    {{.AnnotationKey}}: {{.AnnotationValue}}
spec:
  type: ClusterIP
  selector:
    {{.LabelKey}}: {{.LabelValue}}
  ports:
    - name: http
      port: 80
      protocol: TCP
      targetPort: http
`
