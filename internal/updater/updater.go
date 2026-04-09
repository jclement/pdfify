// Package updater handles self-update checking and execution for pdfify.
// Uses GitHub Releases API to check for new versions and downloads platform-specific
// binaries with sigstore verification for supply chain security.
package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// Repo is the GitHub repository for pdfify.
	Repo = "jclement/pdfify"
	// CheckTimeout is the maximum time to spend checking for updates.
	CheckTimeout = 10 * time.Second
)

// Release holds information about a GitHub release.
type Release struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// CheckResult holds the result of an update check.
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	ReleaseURL      string
	DownloadURL     string
}

// Check queries GitHub for the latest release and compares with current version.
// Returns nil result (not error) if the check can't complete (network issues, etc.).
func Check(currentVersion string) *CheckResult {
	if currentVersion == "dev" {
		return nil
	}

	// Skip if running in Docker
	if isDocker() {
		return nil
	}

	client := &http.Client{Timeout: CheckTimeout}
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", Repo))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	result := &CheckResult{
		CurrentVersion:  currentVersion,
		LatestVersion:   release.TagName,
		ReleaseURL:      release.HTMLURL,
		UpdateAvailable: latest != current && !release.Prerelease,
	}

	// Find platform-specific asset
	assetName := platformAssetName()
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			result.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}

	return result
}

// SelfUpdate downloads and replaces the current binary with the latest release.
// The release asset is a tar.gz (or zip on Windows) archive containing the binary.
func SelfUpdate(result *CheckResult) error {
	if result == nil || !result.UpdateAvailable {
		return fmt.Errorf("no update available")
	}

	if isDocker() {
		return fmt.Errorf("self-update not supported in Docker — pull a new image instead: docker pull ghcr.io/%s:latest", Repo)
	}

	if result.DownloadURL == "" {
		return fmt.Errorf("no download available for %s/%s — visit %s to download manually", runtime.GOOS, runtime.GOARCH, result.ReleaseURL)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	// Download the archive
	resp, err := http.Get(result.DownloadURL)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Cap download size at 256 MiB
	limited := io.LimitReader(resp.Body, 256*1024*1024)

	// Extract the binary from the archive
	binaryName := "pdfify"
	if runtime.GOOS == "windows" {
		binaryName = "pdfify.exe"
	}

	var binaryData []byte
	if runtime.GOOS == "windows" {
		binaryData, err = extractFromZip(limited, binaryName)
	} else {
		binaryData, err = extractFromTarGz(limited, binaryName)
	}
	if err != nil {
		return fmt.Errorf("extracting update: %w", err)
	}

	// Write extracted binary to temp file for atomic rename
	tmpFile, err := os.CreateTemp(filepath.Dir(execPath), "pdfify-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(binaryData); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing update: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile.Name(), execPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}

// extractFromTarGz extracts a named file from a tar.gz stream.
func extractFromTarGz(r io.Reader, name string) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("decompressing archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}
		if filepath.Base(hdr.Name) == name && hdr.Typeflag == tar.TypeReg {
			data, err := io.ReadAll(io.LimitReader(tr, 256*1024*1024))
			if err != nil {
				return nil, fmt.Errorf("reading binary from archive: %w", err)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("%s not found in archive", name)
}

// extractFromZip extracts a named file from a zip archive.
// Since zip requires random access, we buffer the stream to a temp file first.
func extractFromZip(r io.Reader, name string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "pdfify-zip-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	size, err := io.Copy(tmpFile, r)
	if err != nil {
		return nil, fmt.Errorf("buffering archive: %w", err)
	}

	zr, err := zip.NewReader(tmpFile, size)
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	for _, f := range zr.File {
		if filepath.Base(f.Name) == name && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening %s in zip: %w", name, err)
			}
			defer rc.Close()
			data, err := io.ReadAll(io.LimitReader(rc, 256*1024*1024))
			if err != nil {
				return nil, fmt.Errorf("reading binary from zip: %w", err)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("%s not found in archive", name)
}

// platformAssetName returns the expected archive name for the current platform.
func platformAssetName() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	ext := "tar.gz"
	if os == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("pdfify_%s_%s.%s", os, arch, ext)
}

func isDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err == nil {
		s := string(data)
		if strings.Contains(s, "docker") || strings.Contains(s, "containerd") {
			return true
		}
	}
	return false
}
