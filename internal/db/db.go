package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() (*sql.DB, error) {
	hd, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.New("failed to resolve homedir")
	}
	pathParts := []string{hd, "dns_db"}
	dirPath := path.Join(pathParts...)
	filePath := path.Join(dirPath, "db.sqlite")

	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
