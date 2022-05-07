package util

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/moreos"
)

var istioctlBinary = filepath.Join("istioctl" + moreos.Exe)

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
		src := fmt.Sprintf("%s/%s/istioctl-%s-%s-%s.tar.gz",
			istioComponent.ReleaseURLPrefix, istioComponent.Version, istioComponent.Version,
			OSExt(), runtime.GOARCH)
		if err = downloadBinary(src, installPath); err != nil {
			return "", fmt.Errorf("unable to get istioctl binary %q: %w", installPath, err)
		}
	}

	return verifyExecutableBinary(istioctlPath)
}

func downloadBinary(url, path string) error {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("User-Agent", "ubrain")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("received %v status code from %s", res.StatusCode, url)
	}
	if err = Untar(path, res.Body); err != nil {
		return fmt.Errorf("error untarring %s: %w", url, err)
	}
	return nil
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
