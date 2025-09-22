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

package kubeedge

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/plugin/join/kubeedge"
)

var pluginArgs = &kubeedge.JoinArgs{}

func NewCmd(opts *generic.Options) *cobra.Command {
	joinCmd := &cobra.Command{
		Use:                   "edge",
		Short:                 "Registers a node to kubedge.",
		DisableFlagsInUseLine: true,
		PreRunE: func(c *cobra.Command, args []string) error {
			if pluginArgs.Cluster == "" {
				return fmt.Errorf("please provide cluster")
			}

			if pluginArgs.EdgeNode.IP == "" {
				return fmt.Errorf("please provide the IP of edge node")
			}

			if pluginArgs.CloudCoreAddress == "" {
				return fmt.Errorf("please provide the address of cloudcore")
			}

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			plugin, err := kubeedge.NewJoinPlugin(opts, pluginArgs)
			if err != nil {
				logrus.Errorf("join edge init error: %v", err)
				return fmt.Errorf("join edge init error: %v", err)
			}

			logrus.Debugf("start join KubeEdge Node")
			if err := plugin.Execute(args, nil); err != nil {
				logrus.Errorf("join edge execute error: %v", err)
				return fmt.Errorf("join edge execute error: %v", err)
			}

			return nil
		},
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}

	addFlags(joinCmd)

	return joinCmd
}

func addFlags(cmd *cobra.Command) {
	pflags := cmd.PersistentFlags()
	pflags.StringVar(&pluginArgs.Cluster, "cluster", "",
		"cluster indicates the cluster in which Kubeedge is installed")

	pflags.StringVar(&pluginArgs.CGroupDriver, "cgroup-driver", "cgroupfs",
		"CGroupDriver that uses to manipulate cgroups on the host (cgroupfs or systemd), the default value is cgroupfs")

	pflags.StringVar(&pluginArgs.CertPath, "cert-path", "/etc/kubeedge/certs",
		fmt.Sprintf("The certPath used by edgecore, the default value is %s", "/etc/kubeedge/certs"))

	pflags.StringVar(&pluginArgs.CertPort, "cert-port", "",
		"The port where to apply for the edge certificate")

	pflags.StringVar(&pluginArgs.CloudCoreAddress, "cloudcore-address", "",
		"IP:Port address of KubeEdge CloudCore, will try to get it from cluster if unset")

	pflags.StringSliceVar(&pluginArgs.Labels, "labels", nil,
		`use this key to set the customized labels for node. you can input customized labels like key1=value1,key2=value2`)

	pflags.StringVar(&pluginArgs.EdgeNode.Name, "node-name", "",
		"KubeEdge Node unique identification string, If flag not used then the command will generate a unique id on its own")

	pflags.StringVar(&pluginArgs.EdgeNode.IP, "node-ip", "",
		"KubeEdge Node IP")

	pflags.Uint32Var(&pluginArgs.EdgeNode.Port, "node-port", 22,
		"KubeEdge Node port for SSH, the default values is 22")

	pflags.StringVarP(&pluginArgs.EdgeNode.UserName, "node-username", "u", "root",
		"KubeEdge Node username, the default value is root")

	// TODO: support identity file
	pflags.StringVarP(&pluginArgs.EdgeNode.Password, "node-password", "p", "",
		"KubeEdge Node password, the default value is root")
}
