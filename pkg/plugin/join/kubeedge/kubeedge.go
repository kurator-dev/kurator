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

package kubeedge

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/moreos"
	"kurator.dev/kurator/pkg/util"
)

var (
	keadmBinary             = filepath.Join("keadm" + moreos.Exe)
	kubeedgeNamespace       = "kubeedge"
	kubeedgeTokenSecretName = "tokensecret"
	cloudCoreELBSvcName     = "cloudcore-elb"
)

type JoinArgs struct {
	Cluster          string
	CloudCoreAddress string
	EdgeNode         EdgeNode
	CertPath         string
	CertPort         string
	CGroupDriver     string
	Labels           []string
}

type EdgeNode struct {
	Name     string
	IP       string
	Port     uint32
	UserName string
	Password string
}

type JoinPlugin struct {
	*client.Client

	joinArgs *JoinArgs
	options  *generic.Options

	keadm string
}

func NewJoinPlugin(opts *generic.Options, args *JoinArgs) (*JoinPlugin, error) {
	plugin := &JoinPlugin{
		joinArgs: args,
		options:  opts,
		keadm:    "/usr/local/bin/keadm",
	}

	rest := opts.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	plugin.Client = c

	return plugin, nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (p *JoinPlugin) Execute(cmdArgs, environment []string) error {
	if err := p.precheck(); err != nil {
		return err
	}

	clusterClient, err := p.Client.NewClusterClientSet(p.joinArgs.Cluster)
	if err != nil {
		return err
	}

	if err := p.ensureCloudcoreAddress(clusterClient); err != nil {
		return err
	}

	// download keadm
	keadmPath, err := p.installKeadm()
	if err == nil {
		p.keadm = keadmPath
	}

	token, err := p.getEdgeToken(clusterClient)
	if err != nil {
		return err
	}

	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", p.joinArgs.EdgeNode.IP, p.joinArgs.EdgeNode.Port), &ssh.ClientConfig{
		User:            p.joinArgs.EdgeNode.UserName,
		Auth:            []ssh.AuthMethod{ssh.Password(p.joinArgs.EdgeNode.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return fmt.Errorf("SSH dial error: %w", err)
	}

	logrus.Infof("scp keadm to edge node")
	if err := p.scpKeadm(sshClient); err != nil {
		return fmt.Errorf("run scp failed, %w", err)
	}

	logrus.Infof("node %s join edge", p.joinArgs.EdgeNode.IP)
	if err := p.runKeadmJoin(sshClient, token); err != nil {
		logrus.Errorf("failed to join KubeEdge node, %s", err)
		return err
	}

	logrus.Infof("node join edge success")
	return nil
}

func (p *JoinPlugin) getEdgeToken(clusterClient kubeclient.Interface) (string, error) {
	s, err := clusterClient.CoreV1().Secrets(kubeedgeNamespace).Get(context.TODO(), kubeedgeTokenSecretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get edge token, please verify installation, %w", err)
	}
	return string(s.Data["tokendata"]), nil
}

func (p *JoinPlugin) ensureCloudcoreAddress(clusterClient kubeclient.Interface) error {
	if p.joinArgs.CloudCoreAddress != "" {
		return nil
	}

	svc, err := clusterClient.CoreV1().Services(kubeedgeNamespace).Get(context.TODO(), cloudCoreELBSvcName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cloudcore-elb service, %w", err)
	}

	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP == "" {
				continue
			}

			p.joinArgs.CloudCoreAddress = fmt.Sprintf("%s:1000", ingress.IP)
		}
	}

	return fmt.Errorf("failed to get cloudcore address")
}

func (p *JoinPlugin) precheck() error {
	return util.IsClustersReady(p.KarmadaClient(), p.joinArgs.Cluster)
}

func (p *JoinPlugin) scpKeadm(sshClient *ssh.Client) error {
	s, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer s.Close()
	destPath := "/usr/local/bin/" + keadmBinary

	// TODO: find a better way, check sha256sum?
	if err := s.Run("ls -l " + destPath); err == nil {
		logrus.Infof("file exists, skipping scp")
		return nil
	}

	client, err := sftp.NewClient(sshClient)
	if err != nil {
		return fmt.Errorf("creating new SFTP session from existing connection failed: %w", err)
	}
	defer client.Close()

	f, err := os.Open(p.keadm)
	if err != nil {
		return fmt.Errorf("open file fail, %w", err)
	}
	defer f.Close()

	dstFile, err := client.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, f); err != nil {
		return fmt.Errorf("failed to copy file to remote server: %w", err)
	}

	if err := dstFile.Chmod(0755); err != nil {
		return fmt.Errorf("failed to change file permissions on remote server: %w", err)
	}

	return nil
}

func (p *JoinPlugin) runKeadmJoin(sshClient *ssh.Client, token string) error {
	session, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("new session error: %w", err)
	}
	defer session.Close()

	joinArgs := []string{
		"/usr/local/bin/" + keadmBinary,
		"beta",
		"join",
		fmt.Sprintf("--token=%s", token),
		fmt.Sprintf("--cloudcore-ipport=%s", p.joinArgs.CloudCoreAddress),
		fmt.Sprintf("--certPath=%s", p.joinArgs.CertPath),
		fmt.Sprintf("--cgroupdriver=%s", p.joinArgs.CGroupDriver),
	}

	if p.joinArgs.CertPort != "" {
		joinArgs = append(joinArgs, fmt.Sprintf("--certport=%s", p.joinArgs.CertPort))
	}

	if p.joinArgs.EdgeNode.Name != "" {
		joinArgs = append(joinArgs, fmt.Sprintf("--edgenode-name=%s", p.joinArgs.EdgeNode.Name))
	}

	for _, l := range p.joinArgs.Labels {
		joinArgs = append(joinArgs, fmt.Sprintf("--labels=%s", l))
	}

	cmdStr := strings.Join(joinArgs, " ")

	logrus.Infof("run %s", cmdStr)
	return session.Run(cmdStr)
}

func (p *JoinPlugin) installKeadm() (string, error) {
	kubeedgeComponent := p.options.Components["kubeedge"]
	ver := kubeedgeComponent.Version
	if !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}

	installPath := filepath.Join(p.options.HomeDir, kubeedgeComponent.Name, kubeedgeComponent.Version)
	keadmPath := filepath.Join(installPath, fmt.Sprintf("keadm-%s-%s-%s/keadm", ver, util.OSExt(), runtime.GOARCH), keadmBinary)
	_, err := os.Stat(keadmPath)
	if err == nil {
		return keadmPath, nil
	}

	if os.IsNotExist(err) {
		if err = os.MkdirAll(installPath, 0o750); err != nil {
			return "", fmt.Errorf("unable to create directory %q: %w", installPath, err)
		}

		// https://github.com/kubeedge/kubeedge/releases/download/v1.9.2/keadm-v1.9.2-linux-amd64.tar.gz
		url, _ := util.JoinUrlPath(kubeedgeComponent.ReleaseURLPrefix, ver,
			fmt.Sprintf("keadm-%s-%s-%s.tar.gz", ver, util.OSExt(), runtime.GOARCH))
		if _, err = util.DownloadResource(url, installPath); err != nil {
			return "", fmt.Errorf("unable to get keadm binary %q: %w", url, err)
		}
	}
	return util.VerifyExecutableBinary(keadmPath)
}
