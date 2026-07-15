package downloader

import (
	"context"
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

type ProgressFunc func(completed, total int)

type Downloader struct {
	workers    int
	overwrite  bool
	minSize    int64
	maxSize    int64
	maxRetries int
	retryDelay time.Duration
	progressFn ProgressFunc
	client     *http.Client
	rateLimiter *DomainRateLimiter
}

func NewDownloader(workers int, overwrite bool, minSize, maxSize int64) *Downloader {
	return &Downloader{
		workers:    workers,
		overwrite:  overwrite,
		minSize:    minSize,
		maxSize:    maxSize,
		maxRetries: 2,
		retryDelay: 2 * time.Second,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: NewDomainRateLimiter(2, 3),
	}
}

func (d *Downloader) WithRetries(maxRetries int, delay time.Duration) *Downloader {
	d.maxRetries = maxRetries
	d.retryDelay = delay
	return d
}

func (d *Downloader) WithProgress(fn ProgressFunc) *Downloader {
	d.progressFn = fn
	return d
}

func (d *Downloader) WithRateLimit(rps float64, burst int) *Downloader {
	d.rateLimiter = NewDomainRateLimiter(rps, burst)
	return d
}

func (d *Downloader) Run(ctx context.Context, tasks []DownloadTask) []DownloadResult {
	var wg sync.WaitGroup
	taskChan := make(chan DownloadTask, len(tasks))
	results := make([]DownloadResult, 0, len(tasks))
	var mu sync.Mutex
	total := len(tasks)
	var completed int

	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				select {
				case <-ctx.Done():
					mu.Lock()
					results = append(results, DownloadResult{Task: task, Error: ctx.Err()})
					mu.Unlock()
					continue
				default:
				}

				domain := extractDomain(task.URL)
				if err := d.rateLimiter.Wait(domain); err != nil {
					mu.Lock()
					results = append(results, DownloadResult{Task: task, Error: err})
					mu.Unlock()
					continue
				}

				result := d.downloadFileWithRetry(task, ctx)
				mu.Lock()
				results = append(results, result)
				completed++
				if d.progressFn != nil {
					d.progressFn(completed, total)
				}
				mu.Unlock()
			}
		}()
	}

	for _, task := range tasks {
		select {
		case taskChan <- task:
		case <-ctx.Done():
			close(taskChan)
			wg.Wait()
			return results
		}
	}
	close(taskChan)
	wg.Wait()

	return results
}

func (d *Downloader) downloadFileWithRetry(task DownloadTask, ctx context.Context) DownloadResult {
	var lastErr error
	for attempt := 0; attempt <= d.maxRetries; attempt++ {
		if attempt > 0 {
			delay := d.retryDelay * time.Duration(attempt)
			select {
			case <-ctx.Done():
				return DownloadResult{Task: task, Error: ctx.Err()}
			case <-time.After(delay):
			}
		}

		result := d.downloadFile(task)
		if result.Error == nil {
			return result
		}
		lastErr = result.Error
		zap.L().Warn("Download retry",
			zap.String("url", task.URL),
			zap.Int("attempt", attempt+1),
			zap.Error(lastErr),
		)
	}
	return DownloadResult{Task: task, Error: fmt.Errorf("download failed after %d retries: %w", d.maxRetries, lastErr)}
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

	req, err := http.NewRequest(http.MethodGet, task.URL, nil)
	if err != nil {
		return DownloadResult{Task: task, Error: fmt.Errorf("create request: %w", err)}
	}

	if d.overwrite {
		if fi, err := os.Stat(destPath); err == nil {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", fi.Size()))
		}
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return DownloadResult{Task: task, Error: fmt.Errorf("http get: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return DownloadResult{Task: task, Error: fmt.Errorf("http %d", resp.StatusCode)}
	}

	if d.maxSize > 0 && resp.ContentLength > d.maxSize {
		return DownloadResult{Task: task, Error: fmt.Errorf("file too large: %d bytes", resp.ContentLength)}
	}
	if d.minSize > 0 && resp.ContentLength < d.minSize {
		return DownloadResult{Task: task, Error: fmt.Errorf("file too small: %d bytes", resp.ContentLength)}
	}

	openFlags := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == http.StatusPartialContent {
		openFlags |= os.O_APPEND
	} else {
		openFlags |= os.O_TRUNC
	}

	f, err := os.OpenFile(destPath, openFlags, 0644)
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
