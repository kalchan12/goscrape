package storage

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type CrawlRecord struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	URL       string    `gorm:"index;not null" json:"url"`
	Depth     int       `json:"depth"`
	MaxPages  int       `json:"max_pages"`
	PagesHit  int       `json:"pages_hit"`
	Files     int       `json:"files"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DB struct {
	db  *gorm.DB
	dir string
}

func New(dir string) (*DB, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".goscrape")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dir, "history.db")
	gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := gormDB.AutoMigrate(&CrawlRecord{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{db: gormDB, dir: dir}, nil
}

func (d *DB) Insert(record *CrawlRecord) error {
	return d.db.Create(record).Error
}

func (d *DB) Update(record *CrawlRecord) error {
	return d.db.Save(record).Error
}

func (d *DB) List(limit int) ([]CrawlRecord, error) {
	var records []CrawlRecord
	err := d.db.Order("created_at DESC").Limit(limit).Find(&records).Error
	return records, err
}

func (d *DB) GetByID(id uint) (*CrawlRecord, error) {
	var record CrawlRecord
	err := d.db.First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (d *DB) Delete(id uint) error {
	return d.db.Delete(&CrawlRecord{}, id).Error
}

func (d *DB) Clear() error {
	return d.db.Session(&gorm.Session{AllowGlobalUpdate: true}).
		Delete(&CrawlRecord{}).Error
}

func (d *DB) ListByStatus(status string, limit int) ([]CrawlRecord, error) {
	var records []CrawlRecord
	err := d.db.Where("status = ?", status).Order("created_at DESC").Limit(limit).Find(&records).Error
	return records, err
}

type Stats struct {
	TotalCrawls int64   `json:"total_crawls"`
	TotalFiles  int64   `json:"total_files"`
	TotalPages  int64   `json:"total_pages"`
	ByStatus    map[string]int64 `json:"by_status"`
}

func (d *DB) Stats() (*Stats, error) {
	var totalCrawls, totalFiles, totalPages int64
	if err := d.db.Model(&CrawlRecord{}).Count(&totalCrawls).Error; err != nil {
		return nil, err
	}
	if err := d.db.Model(&CrawlRecord{}).Select("COALESCE(SUM(files), 0)").Scan(&totalFiles).Error; err != nil {
		return nil, err
	}
	if err := d.db.Model(&CrawlRecord{}).Select("COALESCE(SUM(pages_hit), 0)").Scan(&totalPages).Error; err != nil {
		return nil, err
	}

	type statusCount struct {
		Status string
		Count  int64
	}
	var rows []statusCount
	if err := d.db.Model(&CrawlRecord{}).Select("status, COUNT(*) as count").Group("status").Find(&rows).Error; err != nil {
		return nil, err
	}
	byStatus := make(map[string]int64)
	for _, r := range rows {
		byStatus[r.Status] = r.Count
	}

	return &Stats{
		TotalCrawls: totalCrawls,
		TotalFiles:  totalFiles,
		TotalPages:  totalPages,
		ByStatus:    byStatus,
	}, nil
}

func (d *DB) ExportJSON(w *os.File) error {
	var records []CrawlRecord
	if err := d.db.Order("created_at DESC").Find(&records).Error; err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(records)
}

func (d *DB) ExportCSV(w *os.File) error {
	var records []CrawlRecord
	if err := d.db.Order("created_at DESC").Find(&records).Error; err != nil {
		return err
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write([]string{"ID", "URL", "Depth", "MaxPages", "PagesHit", "Files", "Status", "CreatedAt", "UpdatedAt"}); err != nil {
		return err
	}

	for _, r := range records {
		if err := writer.Write([]string{
			fmt.Sprintf("%d", r.ID),
			r.URL,
			fmt.Sprintf("%d", r.Depth),
			fmt.Sprintf("%d", r.MaxPages),
			fmt.Sprintf("%d", r.PagesHit),
			fmt.Sprintf("%d", r.Files),
			r.Status,
			r.CreatedAt.Format(time.RFC3339),
			r.UpdatedAt.Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) Cleanup(before time.Time) (int64, error) {
	result := d.db.Where("created_at < ?", before).Delete(&CrawlRecord{})
	return result.RowsAffected, result.Error
}

func (d *DB) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
