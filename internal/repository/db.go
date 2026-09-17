package repository

import (
	"cc-052/internal/config"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func NewDB(cfg *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db, nil
}

func RunMigrations(db *sqlx.DB, migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := db.Exec(string(sql)); err != nil {
			return fmt.Errorf("execute migration %s: %w", f, err)
		}
	}
	return nil
}

func RunSeed(db *sqlx.DB, seedDir string) error {
	files, err := filepath.Glob(filepath.Join(seedDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("list seed files: %w", err)
	}
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read seed %s: %w", f, err)
		}
		if _, err := db.Exec(string(sql)); err != nil {
			return fmt.Errorf("execute seed %s: %w", f, err)
		}
	}
	return nil
}