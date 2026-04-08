// Package server provides the local web server for --watch and --edit modes.
// In watch mode, it serves the PDF in-browser with WebSocket auto-reload on file changes.
// In edit mode, it serves a split-pane UI with a CodeMirror markdown editor and live preview.
package server

import (
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed static/*
var staticFS embed.FS

// Server is the local web server for watch/edit modes.
type Server struct {
	addr       string
	inputPath  string
	outputPath string
	mode       string // "watch" or "edit"

	mu          sync.RWMutex
	pdfData     []byte
	pdfHash     string
	lastModTime time.Time

	// WebSocket clients for reload notifications
	clientsMu sync.Mutex
	clients   map[chan string]bool

	// Callback to trigger reconversion
	onConvert func() error
}

// New creates a new server instance.
func New(inputPath, outputPath, mode string, onConvert func() error) *Server {
	return &Server{
		inputPath:  inputPath,
		outputPath: outputPath,
		mode:       mode,
		clients:    make(map[chan string]bool),
		onConvert:  onConvert,
	}
}

// Start begins serving on a random available port and returns the URL.
func (s *Server) Start() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("finding available port: %w", err)
	}

	s.addr = ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/pdf", s.handlePDF)
	mux.HandleFunc("/events", s.handleSSE)
	mux.HandleFunc("/api/content", s.handleContent)
	mux.HandleFunc("/api/save", s.handleSave)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/static/", s.handleStatic)

	srv := &http.Server{Handler: mux}

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	// Load initial PDF
	s.reloadPDF()

	return fmt.Sprintf("http://%s", s.addr), nil
}

// NotifyReload tells all connected clients to reload the PDF.
func (s *Server) NotifyReload() {
	s.reloadPDF()
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- "reload":
		default:
		}
	}
}

func (s *Server) reloadPDF() {
	data, err := os.ReadFile(s.outputPath)
	if err != nil {
		return
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	s.mu.Lock()
	s.pdfData = data
	s.pdfHash = hash
	s.lastModTime = time.Now()
	s.mu.Unlock()
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmplName := "watch.html"
	if s.mode == "edit" {
		tmplName = "edit.html"
	}

	data, err := staticFS.ReadFile("static/" + tmplName)
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("page").Parse(string(data))
	if err != nil {
		http.Error(w, "template parse error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, map[string]string{
		"Filename": filepath.Base(s.inputPath),
		"Mode":     s.mode,
	})
}

func (s *Server) handlePDF(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	data := s.pdfData
	hash := s.pdfHash
	s.mu.RUnlock()

	if data == nil {
		http.Error(w, "PDF not ready", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("ETag", hash)
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(data)
}

// handleSSE implements Server-Sent Events for reload notifications.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 1)
	s.clientsMu.Lock()
	s.clients[ch] = true
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, ch)
		s.clientsMu.Unlock()
	}()

	// Send initial connected event
	fmt.Fprintf(w, "data: connected\n\n")
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// handleContent returns the markdown file content (for edit mode).
func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(s.inputPath)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// handleSave saves content back to the markdown file (for edit mode).
func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	var payload struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Normalize line endings
	content := strings.ReplaceAll(payload.Content, "\r\n", "\n")

	if err := os.WriteFile(s.inputPath, []byte(content), 0644); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	// Trigger reconversion
	if s.onConvert != nil {
		go func() {
			if err := s.onConvert(); err != nil {
				slog.Error("reconversion failed", "err", err)
			}
			s.NotifyReload()
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	hash := s.pdfHash
	modTime := s.lastModTime
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"hash":     hash,
		"modified": modTime.Format(time.RFC3339),
		"file":     filepath.Base(s.inputPath),
	})
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	data, err := staticFS.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch {
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	}
	w.Write(data)
}
