// converter.go orchestrates the full markdown-to-PDF pipeline.
// It coordinates: config merging, document analysis, Docker container management,
// conversion script generation, and file I/O.
package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jclement/pdfify/internal/config"
	"github.com/jclement/pdfify/internal/docker"
	"github.com/jclement/pdfify/internal/theme"
)

// Progress is called during conversion to report status to the UI.
type Progress func(step string, pct float64)

// DocInfo contains analysis results for a markdown document.
type DocInfo struct {
	H1Count      int
	FirstH1Text  string
	ImageCount   int
	MermaidCount int
	CalloutCount int
	TableRows    int
	CodeBlocks   int
	Images       []string // relative paths
}

// Result holds the output of a successful conversion.
type Result struct {
	OutputPath string
	Pages      int
	FileSize   int64
}

// Convert performs the full markdown-to-PDF conversion for a single file.
func Convert(inputPath string, cfg *config.Config, progress Progress) (*Result, error) {
	// Resolve input path
	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, fmt.Errorf("resolving input path: %w", err)
	}
	if _, err := os.Stat(absInput); err != nil {
		return nil, fmt.Errorf("input file not found: %s", inputPath)
	}

	// Read content and extract frontmatter
	content, err := os.ReadFile(absInput)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	if progress != nil {
		progress("Parsing frontmatter", 0.05)
	}

	fmCfg, _ := ExtractFrontmatter(string(content))

	// Merge config: defaults < preferences < frontmatter < CLI
	prefs, err := config.LoadPreferences()
	if err != nil {
		prefs = &config.Config{}
	}
	defaults := config.DefaultConfig()

	merged := config.Merge(defaults, prefs)
	merged = config.Merge(merged, fmCfg)
	merged = config.Merge(merged, cfg)

	if err := merged.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Analyze document
	if progress != nil {
		progress("Analyzing document", 0.10)
	}
	info := AnalyzeDocument(string(content))

	// Auto-detect heading structure
	if cfg.NumberFrom == 0 && fmCfg.NumberFrom == 0 {
		if info.H1Count == 1 {
			merged.NumberFrom = 2
			// Use H1 as title if no title set
			if merged.Title == "" && info.FirstH1Text != "" {
				merged.Title = info.FirstH1Text
			}
		} else if info.H1Count > 1 {
			merged.NumberFrom = 1
		}
	}

	// Auto-set date if not specified
	if merged.Date == "" {
		merged.Date = "none" // suppress — user can set via frontmatter/prefs
	}
	if merged.Date == "none" || merged.Date == "false" {
		merged.Date = ""
	}

	// Resolve output path
	outputPath := merged.Output
	if outputPath == "" {
		outputPath = strings.TrimSuffix(absInput, filepath.Ext(absInput)) + ".pdf"
	}
	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, fmt.Errorf("resolving output path: %w", err)
	}

	if merged.Preview {
		// Use a cache directory for preview output. Docker Desktop on macOS
		// cannot reliably bind mount os.TempDir() (/var/folders/...) paths
		// back to the host, but ~/Library/Caches is under /Users/ and works.
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			cacheDir = filepath.Dir(absInput) // fallback to input dir
		}
		previewDir := filepath.Join(cacheDir, "pdfify", "preview")
		if err := os.MkdirAll(previewDir, 0755); err != nil {
			return nil, fmt.Errorf("creating preview dir: %w", err)
		}
		base := strings.TrimSuffix(filepath.Base(absInput), filepath.Ext(absInput))
		absOutput = filepath.Join(previewDir, base+".pdf")
	}

	// Get theme
	t := theme.Get(merged.Theme)
	if t == nil {
		return nil, fmt.Errorf("unknown theme: %s", merged.Theme)
	}

	// Ensure Docker image is ready
	if progress != nil {
		progress("Checking Docker image", 0.15)
	}

	// Generate conversion script
	if progress != nil {
		progress("Generating conversion script", 0.20)
	}
	script := GenerateScript(merged, t)

	// Write script to temp file in input directory
	inputDir := filepath.Dir(absInput)
	outputDir := filepath.Dir(absOutput)
	scriptPath := filepath.Join(inputDir, ".pdfify-convert.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return nil, fmt.Errorf("writing conversion script: %w", err)
	}
	defer os.Remove(scriptPath)

	// Build env vars for Docker
	hideFirstH1 := "0"
	if info.H1Count == 1 && cfg.NumberFrom == 0 && fmCfg.NumberFrom == 0 {
		hideFirstH1 = "1"
	}

	env := map[string]string{
		"HIDE_FIRST_H1": hideFirstH1,
	}

	// Run Docker
	if progress != nil {
		progress("Converting with pandoc + XeLaTeX", 0.30)
	}

	inputFile := filepath.Base(absInput)
	outputFile := filepath.Base(absOutput)
	args := []string{"/work/.pdfify-convert.sh", inputFile, "/output/" + outputFile}

	output, err := docker.Run(inputDir, outputDir, env, args)
	if err != nil {
		return nil, fmt.Errorf("conversion failed: %w\nOutput: %s", err, output)
	}

	if progress != nil {
		progress("Done", 1.0)
	}

	// Get output file info
	fi, err := os.Stat(absOutput)
	if err != nil {
		return nil, fmt.Errorf("output file not found after conversion: %w", err)
	}

	return &Result{
		OutputPath: absOutput,
		FileSize:   fi.Size(),
	}, nil
}

// AnalyzeDocument scans markdown content for structural elements.
func AnalyzeDocument(content string) *DocInfo {
	info := &DocInfo{}
	inCode := false

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "```") {
			inCode = !inCode
			if strings.HasPrefix(line, "```mermaid") {
				info.MermaidCount++
			}
			if !inCode && !strings.HasPrefix(line, "```mermaid") {
				info.CodeBlocks++
			}
			continue
		}

		if inCode {
			continue
		}

		// Count H1 headings
		if strings.HasPrefix(line, "# ") {
			info.H1Count++
			if info.H1Count == 1 {
				info.FirstH1Text = strings.TrimPrefix(line, "# ")
			}
		}

		// Count callouts
		if strings.HasPrefix(line, "> [!") {
			info.CalloutCount++
		}

		// Count table rows
		if strings.HasPrefix(line, "|") {
			info.TableRows++
		}

		// Find image references
		imgRe := regexp.MustCompile(`!\[[^\]]*\]\(([^)]+)\)`)
		matches := imgRe.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 && !strings.HasPrefix(m[1], "http") {
				info.Images = append(info.Images, m[1])
				info.ImageCount++
			}
		}
	}

	// Code blocks count is pairs of ```, minus mermaid pairs
	info.CodeBlocks = (info.CodeBlocks - info.MermaidCount)
	if info.CodeBlocks < 0 {
		info.CodeBlocks = 0
	}

	return info
}
