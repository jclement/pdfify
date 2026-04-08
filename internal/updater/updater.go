// Package updater handles self-update checking and execution for pdfify.
// Uses GitHub Releases API to check for new versions and downloads platform-specific
// binaries with sigstore verification for supply chain security.
package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

	// Download new binary
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

	// Write to temp file in same directory as current binary (for atomic rename)
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	tmpFile, err := os.CreateTemp(os.TempDir(), "pdfify-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, limited); err != nil {
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
