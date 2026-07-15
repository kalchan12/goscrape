package downloader

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDownloader(t *testing.T) {
	d := NewDownloader(3, false, 0, 0)
	assert.NotNil(t, d)
	assert.Equal(t, 3, d.workers)
	assert.False(t, d.overwrite)
}

func TestRunEmptyTasks(t *testing.T) {
	d := NewDownloader(3, false, 0, 0)
	results := d.Run(context.Background(), nil)
	assert.Empty(t, results)
}

func TestRunDownloadSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test file content"))
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "goscrape-dl-*")
	defer os.RemoveAll(tmpDir)

	d := NewDownloader(1, false, 0, 0)
	results := d.Run(context.Background(), []DownloadTask{{
		URL:      srv.URL + "/test.txt",
		Filename: "test.txt",
		Filetype: "txt",
		DestDir:  tmpDir,
	}})

	require.Len(t, results, 1)
	assert.NoError(t, results[0].Error)
	assert.False(t, results[0].Skipped)
	assert.Equal(t, int64(17), results[0].Size)
	assert.NotEmpty(t, results[0].MD5)

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "txt", "test.txt"))
	require.NoError(t, err)
	assert.Equal(t, "test file content", string(content))
}

func TestRunDownloadHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	d := NewDownloader(1, false, 0, 0)
	results := d.Run(context.Background(), []DownloadTask{{
		URL:      srv.URL + "/missing",
		Filename: "missing.txt",
		Filetype: "txt",
		DestDir:  os.TempDir(),
	}})

	require.Len(t, results, 1)
	assert.Error(t, results[0].Error)
}

func TestRunDownloadMaxSizeExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("small"))
	}))
	defer srv.Close()

	d := NewDownloader(1, false, 0, 50)
	results := d.Run(context.Background(), []DownloadTask{{
		URL:      srv.URL + "/file.txt",
		Filename: "file.txt",
		Filetype: "txt",
		DestDir:  os.TempDir(),
	}})

	require.Len(t, results, 1)
	assert.Error(t, results[0].Error)
	assert.Contains(t, results[0].Error.Error(), "too large")
}

func TestRunDownloadMinSizeNotMet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "5")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	d := NewDownloader(1, false, 10, 0)
	results := d.Run(context.Background(), []DownloadTask{{
		URL:      srv.URL + "/file.txt",
		Filename: "file.txt",
		Filetype: "txt",
		DestDir:  os.TempDir(),
	}})

	require.Len(t, results, 1)
	assert.Error(t, results[0].Error)
	assert.Contains(t, results[0].Error.Error(), "too small")
}

func TestRunSkipExisting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("content"))
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "goscrape-dl-*")
	defer os.RemoveAll(tmpDir)

	// Create the file first
	dest := filepath.Join(tmpDir, "txt", "existing.txt")
	os.MkdirAll(filepath.Dir(dest), 0755)
	os.WriteFile(dest, []byte("existing content"), 0644)

	d := NewDownloader(1, false, 0, 0)
	results := d.Run(context.Background(), []DownloadTask{{
		URL:      srv.URL + "/existing.txt",
		Filename: "existing.txt",
		Filetype: "txt",
		DestDir:  tmpDir,
	}})

	require.Len(t, results, 1)
	assert.True(t, results[0].Skipped)
	assert.NoError(t, results[0].Error)

	// File should still have original content
	content, _ := os.ReadFile(dest)
	assert.Equal(t, "existing content", string(content))
}

func TestRunOverwrite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("new content"))
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "goscrape-dl-*")
	defer os.RemoveAll(tmpDir)

	// Create the file first
	dest := filepath.Join(tmpDir, "txt", "overwrite.txt")
	os.MkdirAll(filepath.Dir(dest), 0755)
	os.WriteFile(dest, []byte("old content"), 0644)

	d := NewDownloader(1, true, 0, 0) // overwrite=true
	results := d.Run(context.Background(), []DownloadTask{{
		URL:      srv.URL + "/overwrite.txt",
		Filename: "overwrite.txt",
		Filetype: "txt",
		DestDir:  tmpDir,
	}})

	require.Len(t, results, 1)
	assert.False(t, results[0].Skipped)
	assert.NoError(t, results[0].Error)

	// File should now have new content
	content, _ := os.ReadFile(dest)
	assert.Equal(t, "new content", string(content))
}

func TestRunConcurrentDownloads(t *testing.T) {
	var (
		count int
		mu    sync.Mutex
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		c := count
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "file %d", c)
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "goscrape-dl-*")
	defer os.RemoveAll(tmpDir)

	d := NewDownloader(5, false, 0, 0)
	tasks := make([]DownloadTask, 10)
	for i := range tasks {
		tasks[i] = DownloadTask{
			URL:      srv.URL + fmt.Sprintf("/file%d.txt", i),
			Filename: fmt.Sprintf("file%d.txt", i),
			Filetype: "txt",
			DestDir:  tmpDir,
		}
	}

	results := d.Run(context.Background(), tasks)
	assert.Len(t, results, 10)

	var successCount int
	for _, r := range results {
		if r.Error == nil && !r.Skipped {
			successCount++
		}
	}
	assert.Equal(t, 10, successCount)
}

func TestDownloadMD5Calculation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	tmpDir, _ := os.MkdirTemp("", "goscrape-dl-*")
	defer os.RemoveAll(tmpDir)

	d := NewDownloader(1, false, 0, 0)
	results := d.Run(context.Background(), []DownloadTask{{
		URL:      srv.URL + "/hello.txt",
		Filename: "hello.txt",
		Filetype: "txt",
		DestDir:  tmpDir,
	}})

	require.Len(t, results, 1)
	// MD5 of "hello" is 5d41402abc4b2a76b9719d911017c592
	assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", results[0].MD5)
}
