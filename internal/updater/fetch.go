package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DownloadAndExtract fetches a release asset and returns the extracted
// binary path (caller removes it). Supports .tar.gz (unix) and .zip.
func DownloadAndExtract(ctx context.Context, client *http.Client, url, goos string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: http %s", resp.Status)
	}
	dir, err := os.MkdirTemp("", "vpn-update")
	if err != nil {
		return "", err
	}
	name := "vpn"
	if goos == "windows" {
		name += ".exe"
	}
	if strings.HasSuffix(url, ".zip") {
		return unzipTo(resp.Body, dir, name)
	}
	return untarTo(resp.Body, dir, name)
}

func untarTo(r io.Reader, dir, want string) (string, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(hdr.Name) != want || !hdr.FileInfo().Mode().IsRegular() {
			continue
		}
		out := filepath.Join(dir, want)
		f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", err
		}
		_, err = io.Copy(f, tr)
		_ = f.Close()
		if err != nil {
			return "", err
		}
		return out, nil
	}
	return "", fmt.Errorf("binary %s not found in archive", want)
}

func unzipTo(r io.Reader, dir, want string) (string, error) {
	tmp, err := os.CreateTemp("", "vpn-*.zip")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return "", err
	}
	_ = tmp.Close()
	zr, err := zip.OpenReader(tmpName)
	if err != nil {
		return "", err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if filepath.Base(f.Name) != want || f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		out := filepath.Join(dir, want)
		dst, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			_ = rc.Close()
			return "", err
		}
		_, err = io.Copy(dst, rc)
		_ = dst.Close()
		_ = rc.Close()
		if err != nil {
			return "", err
		}
		return out, nil
	}
	return "", fmt.Errorf("binary %s not found in archive", want)
}
