package storage

import (
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

func (d *DB) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
