// Package docker manages the pdfify Docker image lifecycle.
// The image contains pandoc, XeLaTeX, mermaid-cli, Chromium, and fonts — everything
// needed to convert markdown to PDF. The image is built on first use and cached.
// Smart inspection detects when the image needs rebuilding (e.g., version mismatch).
package docker

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os/exec"
	"strings"
)

const (
	// ImageName is the Docker image tag used for the conversion container.
	ImageName = "pdfify"
	// ImageLabelVersion is the label key used to track the pdfify version that built the image.
	ImageLabelVersion = "dev.pdfify.version"
)

//go:embed Dockerfile
var dockerfile string

// Status describes the current state of the Docker image.
type Status int

const (
	StatusReady    Status = iota // Image exists and is current
	StatusMissing                // Image doesn't exist
	StatusOutdated               // Image exists but was built by a different pdfify version
)

// Inspect checks the state of the pdfify Docker image.
// Returns the status and an optional detail message.
func Inspect(appVersion string) (Status, string, error) {
	if !IsDockerAvailable() {
		return StatusMissing, "Docker is not available", fmt.Errorf("docker not found in PATH")
	}

	cmd := exec.Command("docker", "image", "inspect", ImageName, "--format", "{{index .Config.Labels \""+ImageLabelVersion+"\"}}")
	out, err := cmd.Output()
	if err != nil {
		return StatusMissing, "Image not found", nil
	}

	label := strings.TrimSpace(string(out))
	if label == "" || label == "<no value>" {
		return StatusOutdated, "Image has no version label (pre-Go rebuild)", nil
	}

	// Compare Dockerfile hash to detect changes
	currentHash := DockerfileHash()
	if label != currentHash {
		return StatusOutdated, fmt.Sprintf("Image hash %s != current %s", label[:8], currentHash[:8]), nil
	}

	return StatusReady, "Image is up to date", nil
}

// IsDockerAvailable returns true if docker is in PATH and responsive.
func IsDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// DockerfileHash returns a SHA-256 hash of the embedded Dockerfile content.
// This is used as the version label to detect when rebuilds are needed.
func DockerfileHash() string {
	h := sha256.Sum256([]byte(dockerfile))
	return fmt.Sprintf("%x", h)
}

// DockerfileContent returns the embedded Dockerfile text.
func DockerfileContent() string {
	return dockerfile
}

// Build builds the pdfify Docker image with progress output sent to the callback.
// The callback receives each line of Docker build output.
func Build(onProgress func(line string)) error {
	hash := DockerfileHash()

	cmd := exec.Command("docker", "build",
		"-t", ImageName,
		"--label", fmt.Sprintf("%s=%s", ImageLabelVersion, hash),
		"-f", "-",
		".",
	)
	cmd.Stdin = strings.NewReader(dockerfile)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting docker build: %w", err)
	}

	buf := make([]byte, 4096)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 && onProgress != nil {
			for _, line := range strings.Split(string(buf[:n]), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					onProgress(line)
				}
			}
		}
		if readErr != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("docker build failed: %w\n%s", err, stderr.String())
	}
	return nil
}

// Remove removes the pdfify Docker image.
func Remove() error {
	cmd := exec.Command("docker", "rmi", ImageName)
	return cmd.Run()
}

// Run executes a command inside the pdfify Docker container.
// inputDir is mounted as /work (read-only), outputDir as /output.
// env is a map of environment variables to pass to the container.
// script is the path to the conversion script (relative to inputDir).
func Run(inputDir, outputDir string, env map[string]string, args []string) (string, error) {
	dockerArgs := []string{
		"run", "--rm",
		"-v", inputDir + ":/work:ro",
		"-v", outputDir + ":/output",
		"--tmpfs", "/tmp:exec",
	}

	for k, v := range env {
		dockerArgs = append(dockerArgs, "-e", k+"="+v)
	}

	dockerArgs = append(dockerArgs, ImageName)
	dockerArgs = append(dockerArgs, args...)

	cmd := exec.Command("docker", dockerArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stderr.String(), fmt.Errorf("docker run failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String() + stderr.String(), nil
}
