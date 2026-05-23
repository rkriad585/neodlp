package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lrstanley/go-ytdlp"
	"github.com/spf13/cobra"

	"neodlp/internal/banner"
	"neodlp/internal/downloader"
)

var (
	servePort int
	serveHost string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the local REST API daemon",
	Long:  `Starts a background HTTP daemon exposing a local JSON API. Allows queueing downloads from browser extensions or scripts.`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 12121, "Port to run the REST API server on")
	serveCmd.Flags().StringVar(&serveHost, "host", "127.0.0.1", "Host binding (use 0.0.0.0 for public access)")
}

type DaemonJob struct {
	ID             string    `json:"job_id"`
	URL            string    `json:"url"`
	Status         string    `json:"status"` // "queued", "downloading", "processing", "finished", "failed"
	Percent        float64   `json:"percent"`
	Speed          string    `json:"speed"`
	ETA            string    `json:"eta"`
	Size           string    `json:"size"`
	Error          string    `json:"error,omitempty"`
	Filename       string    `json:"filename,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	FinishedAt     time.Time `json:"finished_at,omitempty"`
	cancelFn       context.CancelFunc
	downloadedSize int
}

var (
	jobsMap     = make(map[string]*DaemonJob)
	jobsList    = []*DaemonJob{}
	jobsMutex   sync.RWMutex
	jobCounter  int
	jobQueueCh  = make(chan *DaemonJob, 1000)
	daemonStart = time.Now()
)

func runServe(cmd *cobra.Command, args []string) error {
	fmt.Println(banner.String())
	fmt.Printf("\n📡 Starting NeoDLP REST API Daemon...\n")
	fmt.Printf("   Address : http://%s:%d\n", serveHost, servePort)
	fmt.Printf("   Status  : Running... (Press Ctrl+C to stop)\n\n")

	// Start 3 concurrent worker goroutines for background download processing
	for i := 1; i <= 3; i++ {
		go serveWorker(i)
	}

	// Register routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/download", handleDownloadRequest)
	http.HandleFunc("/api/jobs", handleJobs)
	http.HandleFunc("/api/jobs/", handleJobCancelOrSingle) // Matches /api/jobs/<id> and /api/jobs/<id>/cancel

	addr := fmt.Sprintf("%s:%d", serveHost, servePort)
	server := &http.Server{Addr: addr}

	// Graceful shutdown channel
	go func() {
		<-cmd.Context().Done()
		fmt.Println("\n📡 Shutting down REST API Daemon...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

func setupCors(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	setupCors(w, r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>NeoDLP Daemon</title>
			<style>
				body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background-color: #0f0f15; color: #e5e7eb; padding: 2rem; }
				.card { background: #181824; border-radius: 8px; padding: 1.5rem; max-width: 600px; margin: 0 auto; box-shadow: 0 4px 6px rgba(0,0,0,0.3); border-top: 4px solid #7c3aed; }
				h1 { color: #ffffff; font-size: 1.5rem; margin-top: 0; }
				.btn { background: #7c3aed; color: white; padding: 0.5rem 1rem; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; display: inline-block; }
				.btn:hover { background: #6d28d9; }
				code { background: #2e2e3f; padding: 0.2rem 0.4rem; border-radius: 4px; font-family: monospace; color: #a78bfa; }
			</style>
		</head>
		<body>
			<div class="card">
				<h1>📡 NeoDLP Daemon Mode</h1>
				<p>The REST API daemon is running and healthy on this address.</p>
				<p>Active endpoints:</p>
				<ul>
					<li><code>GET /api/health</code> - Status check</li>
					<li><code>GET /api/jobs</code> - List active & completed downloads</li>
					<li><code>POST /api/download</code> - Request a new download</li>
				</ul>
				<p>Integrate with browser extensions or local automation triggers using standard HTTP POST.</p>
			</div>
		</body>
		</html>
	`))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	setupCors(w, r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"uptime": time.Since(daemonStart).String(),
		"engine": "neodlp daemon v0.2.1",
	})
}

type DownloadRequest struct {
	URL           string `json:"url"`
	Quality       string `json:"quality"`
	Format        string `json:"format"`
	AudioOnly     bool   `json:"audio_only"`
	EmbedMetadata bool   `json:"embed_metadata"`
	Upload        string `json:"upload"` // "telegram", "discord", "custom" or empty
}

func handleDownloadRequest(w http.ResponseWriter, r *http.Request) {
	if setupCors(w, r) {
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var req DownloadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON structure", http.StatusBadRequest)
		return
	}

	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	jobsMutex.Lock()
	jobCounter++
	id := fmt.Sprintf("job_%04d", jobCounter)

	job := &DaemonJob{
		ID:        id,
		URL:       req.URL,
		Status:    "queued",
		Percent:   0.0,
		CreatedAt: time.Now(),
	}

	jobsMap[id] = job
	jobsList = append(jobsList, job)
	jobsMutex.Unlock()

	// Spin up worker task config from request options
	go func() {
		// Prepare Options for downloader
		opts := downloader.Options{
			Quality:       req.Quality,
			Format:        req.Format,
			AudioOnly:     req.AudioOnly,
			EmbedMetadata: req.EmbedMetadata,
			UploadTarget:  req.Upload,
		}
		// Push job with options in context
		jobQueueCh <- job
		// We'll bind options to it once worker pulls it
		_ = opts
	}()

	// Save options on a background mapping
	// Map job to its specific download options
	setJobOptions(id, req)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":  id,
		"status":  "queued",
		"message": "Download task added to queue successfully",
	})
}

var (
	jobOptionsMap = make(map[string]DownloadRequest)
	optionsMutex  sync.RWMutex
)

func setJobOptions(id string, req DownloadRequest) {
	optionsMutex.Lock()
	jobOptionsMap[id] = req
	optionsMutex.Unlock()
}

func getJobOptions(id string) (DownloadRequest, bool) {
	optionsMutex.RLock()
	req, ok := jobOptionsMap[id]
	optionsMutex.RUnlock()
	return req, ok
}

func handleJobs(w http.ResponseWriter, r *http.Request) {
	if setupCors(w, r) {
		return
	}
	jobsMutex.RLock()
	defer jobsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobsList)
}

func handleJobCancelOrSingle(w http.ResponseWriter, r *http.Request) {
	if setupCors(w, r) {
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id := parts[3]

	jobsMutex.RLock()
	job, exists := jobsMap[id]
	jobsMutex.RUnlock()

	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Route /api/jobs/<id>/cancel
	if len(parts) >= 5 && parts[4] == "cancel" {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		jobsMutex.Lock()
		if job.Status == "downloading" || job.Status == "queued" {
			if job.cancelFn != nil {
				job.cancelFn()
			}
			job.Status = "failed"
			job.Error = "Cancelled by user"
			job.FinishedAt = time.Now()
		}
		jobsMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id": id,
			"status": "cancelled",
		})
		return
	}

	// Just /api/jobs/<id>
	if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func serveWorker(workerID int) {
	for job := range jobQueueCh {
		jobsMutex.RLock()
		isCancelled := job.Status == "failed" && job.Error == "Cancelled by user"
		jobsMutex.RUnlock()

		if isCancelled {
			continue
		}

		req, exists := getJobOptions(job.ID)
		if !exists {
			jobsMutex.Lock()
			job.Status = "failed"
			job.Error = "Options missing"
			job.FinishedAt = time.Now()
			jobsMutex.Unlock()
			continue
		}

		ctx, cancel := context.WithCancel(context.Background())
		jobsMutex.Lock()
		job.Status = "downloading"
		job.cancelFn = cancel
		jobsMutex.Unlock()

		opts := downloader.Options{
			Quality:       req.Quality,
			Format:        req.Format,
			AudioOnly:     req.AudioOnly,
			EmbedMetadata: req.EmbedMetadata,
			UploadTarget:  req.Upload,
		}

		started := time.Now()

		progressFn := func(prog ytdlp.ProgressUpdate) {
			jobsMutex.Lock()
			job.Percent = prog.Percent()
			// Speed
			elapsed := time.Since(started)
			if elapsed > 0 && prog.DownloadedBytes > 0 {
				rate := float64(prog.DownloadedBytes) / elapsed.Seconds()
				job.Speed = formatDaemonBytes(int(rate)) + "/s"
			}
			// ETA
			eta := prog.ETA()
			if eta > 0 && eta < 24*time.Hour {
				if eta >= time.Minute {
					job.ETA = fmt.Sprintf("%dm%ds", int(eta.Minutes()), int(eta.Seconds())%60)
				} else {
					job.ETA = fmt.Sprintf("%ds", int(eta.Seconds()))
				}
			} else {
				job.ETA = ""
			}
			// Size
			if prog.DownloadedBytes > 0 {
				job.Size = formatDaemonBytes(prog.DownloadedBytes)
				if prog.TotalBytes > 0 {
					job.Size += " / " + formatDaemonBytes(prog.TotalBytes)
				}
			}
			if prog.Status == "processing" {
				job.Status = "processing"
			}
			jobsMutex.Unlock()
		}

		results, err := downloader.DownloadWithProgress(ctx, []string{job.URL}, opts, progressFn)

		jobsMutex.Lock()
		if err != nil {
			if ctx.Err() != nil {
				job.Status = "failed"
				job.Error = "Cancelled by user"
			} else {
				job.Status = "failed"
				job.Error = err.Error()
			}
		} else {
			job.Status = "finished"
			job.Percent = 100.0
			if len(results) > 0 {
				job.Filename = filepath.Base(results[0].Filename)
			}
		}
		job.FinishedAt = time.Now()
		jobsMutex.Unlock()
		cancel()
	}
}

func formatDaemonBytes(b int) string {
	if b == 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
