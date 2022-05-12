package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/moreos"
)

var (
	istioctlBinary   = filepath.Join("istioctl" + moreos.Exe)
	karmadactlBinary = filepath.Join("kubectl-karmada" + moreos.Exe)
)

type BinaryGetter struct {
	settings *generic.Options
}

func NewBinaryGetter(o *generic.Options) *BinaryGetter {
	return &BinaryGetter{
		settings: o,
	}
}

func (g *BinaryGetter) Istioctl() (string, error) {
	istioComponent := g.settings.Components["istio"]

	installPath := filepath.Join(g.settings.HomeDir, istioComponent.Name, istioComponent.Version)
	istioctlPath := filepath.Join(installPath, istioctlBinary)
	_, err := os.Stat(istioctlPath)
	if err == nil {
		return istioctlPath, nil
	}

	if os.IsNotExist(err) {
		if err = os.MkdirAll(installPath, 0o750); err != nil {
			return "", fmt.Errorf("unable to create directory %q: %w", installPath, err)
		}
		url := fmt.Sprintf("%s/%s/istioctl-%s-%s-%s.tar.gz",
			istioComponent.ReleaseURLPrefix, istioComponent.Version, istioComponent.Version,
			OSExt(), runtime.GOARCH)
		if _, err := downloadResource(url, installPath); err != nil {
			return "", fmt.Errorf("unable to get istioctl binary %q: %w", installPath, err)
		}
	}

	return verifyExecutableBinary(istioctlPath)
}

func (g *BinaryGetter) Karmadactl() (string, error) {
	karmadaComponent := g.settings.Components["karmada"]

	installPath := filepath.Join(g.settings.HomeDir, karmadaComponent.Name, karmadaComponent.Version)
	karmadactlPath := filepath.Join(installPath, karmadactlBinary)
	_, err := os.Stat(karmadactlPath)
	if err == nil {
		return karmadactlPath, nil
	}

	if os.IsNotExist(err) {
		if err = os.MkdirAll(installPath, 0o750); err != nil {
			return "", fmt.Errorf("unable to create directory %q: %w", installPath, err)
		}
		url := fmt.Sprintf("%s/%s/kubectl-karmada-%s-%s.tgz",
			karmadaComponent.ReleaseURLPrefix, karmadaComponent.Version, OSExt(), runtime.GOARCH)
		if _, err = downloadResource(url, installPath); err != nil {
			return "", fmt.Errorf("unable to get istioctl binary %q: %w", installPath, err)
		}
	}

	return verifyExecutableBinary(karmadactlPath)
}

func (g *BinaryGetter) Valcano() (string, error) {
	volcano := g.settings.Components["volcano"]

	// x84_64 https://raw.githubusercontent.com/volcano-sh/volcano/master/installer/volcano-development.yaml
	// arm64 https://raw.githubusercontent.com/volcano-sh/volcano/v1.5.1/installer/volcano-development.yaml
	ver := volcano.Version
	if ver != "master" && !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}

	var url string
	// TODO: change it, the machine used to install volcano can be different from the destination cluster arch
	switch runtime.GOARCH {
	case "amd64":
		url = fmt.Sprintf("%s%s/installer/volcano-development.yaml", volcano.ReleaseURLPrefix, ver)
	case "arm64":
		url = fmt.Sprintf("%s%s/installer/volcano-development-arm64.yaml", volcano.ReleaseURLPrefix, ver)
	default:
		return "", fmt.Errorf("os arch %s is not supportted", runtime.GOARCH)
	}

	yaml, err := downloadResource(url, "")
	if err != nil {
		return "", err
	}

	return yaml, nil
}

// downloadResource retrieves resources from remote url.
// If path is provided, it will also write it to the dir.
// If the resource is a tar file, it will first untar it.
func downloadResource(url, path string) (raw string, err error) {
	// TODO: make it configurable
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", "ubrain")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received %v status code from %s", res.StatusCode, url)
	}

	// 1. no path provided, maybe this is raw content from github yaml
	if path == "" {
		rawBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("read response body error %s", url)
		}
		return string(rawBytes), nil
	}

	// 2. untar the zip package in to the path
	if strings.HasSuffix(url, ".tgz") || strings.HasSuffix(url, ".tar.gz") {
		if err = Untar(path, res.Body); err != nil {
			return "", fmt.Errorf("error untarring %s: %w", url, err)
		}
		return "", nil
	}

	// 3. write the response file to the path
	strings.Split(url, "/")
	fileName := ""
	out, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("create file %s failed: %v", fileName, err)
	}
	_, err = io.Copy(out, res.Body)
	return "", err
}

func verifyExecutableBinary(binaryPath string) (string, error) {
	stat, err := os.Stat(binaryPath)
	if err != nil {
		return "", err
	}
	if !moreos.IsExecutable(stat) {
		return "", fmt.Errorf("binary not executable at %q", binaryPath)
	}
	return binaryPath, nil
}

func OSExt() string {
	switch runtime.GOOS {
	case "darwin":
		return "osx"
	case "linux":
		return "linux"
	case "windows":
		return "win"
	}

	return ""
}
