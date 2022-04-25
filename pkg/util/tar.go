package util

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Untar unarchives the compressed "src" which is "tar.gz" stream.
func Untar(dst string, src io.Reader) error { // dst, src order like io.Copy
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return err
	}

	zSrc, err := newDecompressor(src)
	if err != nil {
		return err
	}
	defer zSrc.Close() //nolint

	tr := tar.NewReader(zSrc)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		srcPath := filepath.Clean(header.Name)
		slash := strings.Index(srcPath, string(os.PathSeparator))
		if slash == -1 { // strip leading path
			dstPath := filepath.Join(dst, srcPath)
			info := header.FileInfo()
			if err := extractFile(dstPath, tr, info.Mode()); err != nil {
				return err
			}
			continue
		}
		srcPath = srcPath[slash+1:]

		dstPath := filepath.Join(dst, srcPath)
		info := header.FileInfo()

		if info.IsDir() {
			if err := os.MkdirAll(dstPath, info.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := extractFile(dstPath, tr, info.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(dst string, src io.Reader, perm os.FileMode) error {
	file, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm) //nolint:gosec
	if err != nil {
		return err
	}
	defer file.Close() //nolint
	_, err = io.Copy(file, src)
	return err
}

// newDecompressor returns a "gzip" decompression function based on bytes in the stream.
func newDecompressor(r io.Reader) (io.ReadCloser, error) {
	br := bufio.NewReader(r)
	return gzip.NewReader(br)
}
