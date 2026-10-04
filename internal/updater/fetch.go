package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
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
	// Cap decompression output: release assets are ~10MB.
	body := io.LimitReader(resp.Body, 128<<20)
	if strings.HasSuffix(url, ".zip") {
		return unzipTo(body, dir, name)
	}
	return untarTo(body, dir, name)
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
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(hdr.Name) != want || !hdr.FileInfo().Mode().IsRegular() {
			continue
		}
		// basename is pinned to want above; 0755 keeps the binary executable.
		// path built from MkdirTemp dir + fixed binary name (gosec exclusion, see line below).
		out := filepath.Join(dir, want) //nolint:gosec
		// release binary must stay executable (gosec exclusion, see line below).
		f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755) //nolint:gosec
		if err != nil {
			return "", err
		}
		_, err = io.Copy(f, tr) //nolint:gosec // stream capped by 128MB LimitReader above
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
		// path built from MkdirTemp dir + fixed binary name (gosec exclusion, see line below).
		out := filepath.Join(dir, want) //nolint:gosec
		// release binary must stay executable (gosec exclusion, see line below).
		dst, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755) //nolint:gosec
		if err != nil {
			_ = rc.Close()
			return "", err
		}
		_, err = io.Copy(dst, rc) //nolint:gosec // stream capped by 128MB LimitReader above
		_ = dst.Close()
		_ = rc.Close()
		if err != nil {
			return "", err
		}
		return out, nil
	}
	return "", fmt.Errorf("binary %s not found in archive", want)
}
