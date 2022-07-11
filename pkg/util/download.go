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

package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"kurator.dev/kurator/pkg/moreos"
)

// DownloadResource retrieves resources from remote url.
// If path is provided, it will also write it to the dir.
// If the resource is a tar file, it will first untar it.
func DownloadResource(url, path string) (raw string, err error) {
	logrus.Infof("begin to download resource %s -> %s", url, path)
	// TODO: make it configurable
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", "kurator")
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
		if err = Untar(res.Body, path); err != nil {
			return "", fmt.Errorf("error untarring %s: %w", url, err)
		}
		return "", nil
	}

	// 3. write the response file to the path
	out, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create file %s failed: %v", path, err)
	}
	_, err = io.Copy(out, res.Body)
	return "", err
}

func VerifyExecutableBinary(binaryPath string) (string, error) {
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
