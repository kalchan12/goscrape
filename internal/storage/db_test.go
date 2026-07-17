package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	dir, err := os.MkdirTemp("", "goscrape-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	db, err := New(dir)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func insertTestRecord(t *testing.T, db *DB, url string, status string) *CrawlRecord {
	t.Helper()
	record := &CrawlRecord{
		URL:       url,
		Depth:     1,
		MaxPages:  10,
		PagesHit:  5,
		Files:     2,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := db.Insert(record)
	require.NoError(t, err)
	return record
}

func TestNew(t *testing.T) {
	dir, err := os.MkdirTemp("", "goscrape-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	db, err := New(dir)
	require.NoError(t, err)
	defer db.Close()

	assert.DirExists(t, dir)
	dbPath := filepath.Join(dir, "history.db")
	assert.FileExists(t, dbPath)
}

func TestInsert(t *testing.T) {
	db := newTestDB(t)
	record := insertTestRecord(t, db, "https://example.com", "completed")
	assert.NotZero(t, record.ID)
}

func TestGetByID(t *testing.T) {
	db := newTestDB(t)
	original := insertTestRecord(t, db, "https://example.com", "completed")

	fetched, err := db.GetByID(original.ID)
	require.NoError(t, err)
	assert.Equal(t, original.URL, fetched.URL)
	assert.Equal(t, original.Status, fetched.Status)
}

func TestGetByIDNotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetByID(999)
	assert.Error(t, err)
}

func TestList(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, "https://example.com/a", "completed")
	insertTestRecord(t, db, "https://example.com/b", "completed")
	insertTestRecord(t, db, "https://example.com/c", "completed")

	records, err := db.List(10)
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

func TestListLimit(t *testing.T) {
	db := newTestDB(t)
	for i := 0; i < 5; i++ {
		insertTestRecord(t, db, "https://example.com/page", "completed")
	}

	records, err := db.List(2)
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestListOrder(t *testing.T) {
	db := newTestDB(t)
	_ = insertTestRecord(t, db, "https://example.com/first", "completed")
	time.Sleep(10 * time.Millisecond)
	_ = insertTestRecord(t, db, "https://example.com/second", "completed")

	records, err := db.List(10)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/second", records[0].URL)
	assert.Equal(t, "https://example.com/first", records[1].URL)
}

func TestUpdate(t *testing.T) {
	db := newTestDB(t)
	record := insertTestRecord(t, db, "https://example.com", "running")

	record.Status = "completed"
	err := db.Update(record)
	require.NoError(t, err)

	fetched, err := db.GetByID(record.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", fetched.Status)
}

func TestDelete(t *testing.T) {
	db := newTestDB(t)
	record := insertTestRecord(t, db, "https://example.com", "completed")

	err := db.Delete(record.ID)
	require.NoError(t, err)

	_, err = db.GetByID(record.ID)
	assert.Error(t, err)
}

func TestClear(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, "https://example.com/a", "completed")
	insertTestRecord(t, db, "https://example.com/b", "completed")

	err := db.Clear()
	require.NoError(t, err)

	records, err := db.List(10)
	require.NoError(t, err)
	assert.Empty(t, records)
}

func TestClose(t *testing.T) {
	db := newTestDB(t)
	err := db.Close()
	assert.NoError(t, err)
}

func TestListByStatus(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, "https://example.com/a", "completed")
	insertTestRecord(t, db, "https://example.com/b", "running")
	insertTestRecord(t, db, "https://example.com/c", "completed")

	completed, err := db.ListByStatus("completed", 10)
	require.NoError(t, err)
	assert.Len(t, completed, 2)

	running, err := db.ListByStatus("running", 10)
	require.NoError(t, err)
	assert.Len(t, running, 1)
}

func TestListByURL(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, "https://example.com/page1", "completed")
	insertTestRecord(t, db, "https://other.com/page", "completed")
	insertTestRecord(t, db, "https://example.com/page2", "completed")

	results, err := db.ListByURL("example.com", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestListByDateRange(t *testing.T) {
	db := newTestDB(t)
	old := &CrawlRecord{
		URL: "https://example.com/old", Depth: 1, MaxPages: 10, PagesHit: 5, Files: 2, Status: "completed",
		CreatedAt: time.Now().Add(-48 * time.Hour), UpdatedAt: time.Now().Add(-48 * time.Hour),
	}
	require.NoError(t, db.Insert(old))
	insertTestRecord(t, db, "https://example.com/new", "completed")

	start := time.Now().Add(-24 * time.Hour)
	end := time.Now().Add(1 * time.Hour)
	results, err := db.ListByDateRange(start, end, 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "https://example.com/new", results[0].URL)
}

func TestStats(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, "https://example.com/a", "completed")
	insertTestRecord(t, db, "https://example.com/b", "completed")
	insertTestRecord(t, db, "https://example.com/c", "failed")

	stats, err := db.Stats()
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.TotalCrawls)
	assert.Equal(t, int64(6), stats.TotalFiles)
	assert.Equal(t, int64(15), stats.TotalPages)
	assert.Equal(t, int64(2), stats.ByStatus["completed"])
	assert.Equal(t, int64(1), stats.ByStatus["failed"])
}

func TestExportJSON(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, "https://example.com", "completed")

	f, err := os.CreateTemp("", "export-*.json")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	err = db.ExportJSON(f)
	require.NoError(t, err)
	f.Close()

	data, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	assert.Contains(t, string(data), "https://example.com")
}

func TestExportCSV(t *testing.T) {
	db := newTestDB(t)
	insertTestRecord(t, db, "https://example.com", "completed")

	f, err := os.CreateTemp("", "export-*.csv")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	err = db.ExportCSV(f)
	require.NoError(t, err)
	f.Close()

	data, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	assert.Contains(t, string(data), "https://example.com")
	assert.Contains(t, string(data), "ID,URL,Depth")
}

func TestCleanup(t *testing.T) {
	db := newTestDB(t)
	old := &CrawlRecord{
		URL:       "https://example.com/old",
		Depth:     1,
		MaxPages:  10,
		PagesHit:  5,
		Files:     2,
		Status:    "completed",
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-48 * time.Hour),
	}
	require.NoError(t, db.Insert(old))

	newRecord := insertTestRecord(t, db, "https://example.com/new", "completed")

	count, err := db.Cleanup(time.Now().Add(-24 * time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	_, err = db.GetByID(old.ID)
	assert.Error(t, err)

	_, err = db.GetByID(newRecord.ID)
	assert.NoError(t, err)
}
