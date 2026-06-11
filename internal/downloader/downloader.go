package downloader

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

type DownloadTask struct {
	URL      string
	Filename string
	Filetype string
	DestDir  string
}

type DownloadResult struct {
	Task     DownloadTask
	Size     int64
	Duration time.Duration
	MD5      string
	Error    error
	Skipped  bool
}

type Downloader struct {
	workers  int
	overwrite bool
	minSize  int64
	maxSize  int64
	client   *http.Client
}

func NewDownloader(workers int, overwrite bool, minSize, maxSize int64) *Downloader {
	return &Downloader{
		workers:   workers,
		overwrite: overwrite,
		minSize:   minSize,
		maxSize:   maxSize,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (d *Downloader) Run(tasks []DownloadTask) []DownloadResult {
	var wg sync.WaitGroup
	taskChan := make(chan DownloadTask, len(tasks))
	results := make([]DownloadResult, 0, len(tasks))
	var mu sync.Mutex

	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				result := d.downloadFile(task)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		}()
	}

	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)
	wg.Wait()

	return results
}

func (d *Downloader) downloadFile(task DownloadTask) DownloadResult {
	start := time.Now()

	destPath := filepath.Join(task.DestDir, task.Filetype, task.Filename)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return DownloadResult{Task: task, Error: fmt.Errorf("mkdir: %w", err)}
	}

	if !d.overwrite {
		if _, err := os.Stat(destPath); err == nil {
			zap.L().Info("Skipping existing file", zap.String("path", destPath))
			return DownloadResult{Task: task, Skipped: true}
		}
	}

	resp, err := d.client.Get(task.URL)
	if err != nil {
		return DownloadResult{Task: task, Error: fmt.Errorf("http get: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DownloadResult{Task: task, Error: fmt.Errorf("http %d", resp.StatusCode)}
	}

	if d.maxSize > 0 && resp.ContentLength > d.maxSize {
		return DownloadResult{Task: task, Error: fmt.Errorf("file too large: %d bytes", resp.ContentLength)}
	}
	if d.minSize > 0 && resp.ContentLength < d.minSize {
		return DownloadResult{Task: task, Error: fmt.Errorf("file too small: %d bytes", resp.ContentLength)}
	}

	f, err := os.Create(destPath)
	if err != nil {
		return DownloadResult{Task: task, Error: fmt.Errorf("create: %w", err)}
	}

	hash := md5.New()
	writer := io.MultiWriter(f, hash)
	size, err := io.Copy(writer, resp.Body)
	if err != nil {
		f.Close()
		os.Remove(destPath)
		return DownloadResult{Task: task, Error: fmt.Errorf("download: %w", err)}
	}
	f.Close()

	sum := fmt.Sprintf("%x", hash.Sum(nil))
	elapsed := time.Since(start)

	zap.L().Info("Downloaded",
		zap.String("url", task.URL),
		zap.String("dest", destPath),
		zap.Int64("size", size),
		zap.Duration("elapsed", elapsed),
	)

	return DownloadResult{
		Task:     task,
		Size:     size,
		Duration: elapsed,
		MD5:      sum,
	}
}
