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

package tool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gosuri/uitable"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"kurator.dev/kurator/pkg/generic"
)

// OutputFormat is a type for capturing supported output formats
type OutputFormat string

const (
	Table     OutputFormat = "table"
	TableWIDE OutputFormat = "wide"
	JSON      OutputFormat = "json"
	YAML      OutputFormat = "yaml"
)

// Formats returns a list of the string representation of the supported formats
func Formats() []string {
	return []string{Table.String(), TableWIDE.String(), JSON.String(), YAML.String()}
}

func (o OutputFormat) String() string {
	return string(o)
}

type componentElement struct {
	Name             string `yaml:"name"`
	Status           string `yaml:"status"`
	Version          string `yaml:"version"`
	Cli              string `yaml:"cli"`
	Hub              string `yaml:"hub"`
	ReleaseURLPrefix string `yaml:"releaseURLPrefix"`
}

type listOptions struct {
	options    *generic.Options
	components map[string]componentElement
	output     string
}

func NewListCmd(out io.Writer, opts *generic.Options) *cobra.Command {
	o := &listOptions{
		options: opts,
	}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "list tool information",
		Example: getExample(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.complete(args); err != nil {
				return err
			}

			if err := o.validate(); err != nil {
				return err
			}

			return o.Run(out)
		},
	}

	// flags
	f := cmd.PersistentFlags()
	f.StringVarP(&o.output, "output", "o", Table.String(), fmt.Sprintf("Output format. (-o|--output=)%v", Formats()))

	// completion
	outputFlagCompletion(cmd, "output")

	return cmd
}

func getExample() string {
	return `  # List all components Cli tool.
  kurator tool list

  # List the Cli tools for all components and more information (such as tool path).
  kurator tool list -o wide

  # List specified components Cli tool.
  kurator tool list istio karmada

  # List component Cli tools in JSON output format.
  kurator tool list -o json

  # List component Cli tools in YAML output format.
  kurator tool list -o yaml

  # List a single components Cli tool in JSON output format.
  kurator tool list istio -o json`
}

func outputFlagCompletion(cmd *cobra.Command, flag string) {
	if err := cmd.RegisterFlagCompletionFunc(flag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return Formats(), cobra.ShellCompDirectiveDefault
	}); err != nil {
		logrus.Warn(err)
	}
}

func (l *listOptions) complete(args []string) error {
	l.components = make(map[string]componentElement)

	if len(args) != 0 {
		return l.specifiedComponent(args)
	}

	for k, v := range l.options.Components {
		if cliToolName(k) == "" {
			delete(l.options.Components, k)
			continue
		}
		l.add(k, v)
	}

	return nil
}

func (l *listOptions) validate() error {
	if len(l.components) == 0 {
		return fmt.Errorf("components have no cli tool")
	}

	return nil
}

func (l *listOptions) specifiedComponent(args []string) error {
	for _, v := range args {
		if cliToolName(v) == "" {
			continue
		}

		l.add(v, l.options.Components[v])
	}

	return nil
}

func (l *listOptions) add(name string, v generic.Component) {
	status, cliPath := toolsStatus(filepath.Join(l.options.HomeDir, v.Name, v.Version, cliToolName(name)))
	l.components[name] = componentElement{
		Name:             v.Name,
		Status:           status,
		Version:          v.Version,
		Cli:              cliPath,
		Hub:              v.Hub,
		ReleaseURLPrefix: v.ReleaseURLPrefix,
	}
}

func (l *listOptions) Run(out io.Writer) error {
	cs, err := yaml.Marshal(l.components)
	if err != nil {
		return fmt.Errorf("conversion failed. %v", err)
	}

	w := &toolListWriter{
		cs: cs,
	}

	for _, v := range l.components {
		w.entry = append(w.entry, v)
	}
	// sort by component name
	sort.Slice(w.entry, func(i, j int) bool {
		return w.entry[i].Name < w.entry[j].Name
	})

	return w.PrintObj(out, l.output)
}

type toolListWriter struct {
	entry []componentElement
	cs    []byte
}

func (t *toolListWriter) writeTable(out io.Writer) error {
	table := uitable.New()

	table.AddRow("NAME", "VERSION", "STATUS")
	for _, te := range t.entry {
		table.AddRow(te.Name, te.Version, te.Status)
	}

	return t.encodeTable(out, table)
}

func (t *toolListWriter) writeTableWIDE(out io.Writer) error {
	table := uitable.New()

	table.AddRow("NAME", "VERSION", "STATUS", "CLI", "HUB", "RELEASE-URL-PREFIX")
	for _, te := range t.entry {
		table.AddRow(te.Name, te.Version, te.Status, te.Cli, te.Hub, te.ReleaseURLPrefix)
	}

	return t.encodeTable(out, table)
}

func (t *toolListWriter) encodeTable(out io.Writer, table *uitable.Table) error {
	raw := table.Bytes()
	raw = append(raw, []byte("\n")...)
	_, err := out.Write(raw)
	if err != nil {
		return fmt.Errorf("unable to write table output. %v", err)
	}
	return nil
}

func (t *toolListWriter) writeJSON(out io.Writer) error {
	js, err := yaml.YAMLToJSON(t.cs)
	if err != nil {
		return fmt.Errorf("yaml cannot convert json. %v", err)
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, js, "", "\t"); err != nil {
		return fmt.Errorf("can't format json. %v", err)
	}

	if _, err := prettyJSON.WriteTo(out); err != nil {
		return fmt.Errorf("unable to write json output. %v", err)
	}
	return nil
}

func (t *toolListWriter) writeYAML(out io.Writer) error {
	if _, err := out.Write(t.cs); err != nil {
		return err
	}
	return nil
}

func (t *toolListWriter) PrintObj(out io.Writer, format string) error {
	switch strings.ToLower(format) {
	case Table.String():
		return t.writeTable(out)
	case TableWIDE.String():
		return t.writeTableWIDE(out)
	case JSON.String():
		return t.writeJSON(out)
	case YAML.String():
		return t.writeYAML(out)
	}

	return fmt.Errorf("invalid format type")
}

// cliToolName returns the Cli name of the component
// When the newly added component has cli, need to add cli name.
func cliToolName(componentName string) string {
	switch strings.ToLower(componentName) {
	case "istio":
		return "istioctl"
	case "karmada":
		return "kubectl-karmada"
	case "submariner":
		return "subctl"
	case "kubeedge":
		return "keadm"
	case "argocd":
		return "argocd"
	default:
		return ""
	}
}

func toolsStatus(path string) (string, string) {
	if !exists(path) {
		return "NotReady", ""
	}
	return "Ready", path
}

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return os.IsExist(err)
	}
	return true
}
