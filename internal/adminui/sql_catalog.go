package adminui

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultSQLRoot   = "db/sql"
	defaultMaxSQLLen = 256 * 1024
)

// SQLCatalog summarizes schema/query files under db/sql for docs display.
type SQLCatalog struct {
	Root    string    `json:"root"`
	Missing bool      `json:"missing"`
	Error   string    `json:"error,omitempty"`
	Schemas []SQLFile `json:"schemas"`
	Queries []SQLFile `json:"queries"`
}

// SQLFile describes one SQL file surfaced in the docs explorer.
type SQLFile struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	Lines     int    `json:"lines"`
	Truncated bool   `json:"truncated"`
	Content   string `json:"content"`
}

// LoadSQLCatalog reads schema/query SQL files from root (defaults to db/sql).
func LoadSQLCatalog(root string) SQLCatalog {
	root = strings.TrimSpace(root)
	if root == "" {
		root = defaultSQLRoot
	}
	catalog := SQLCatalog{
		Root:    root,
		Schemas: []SQLFile{},
		Queries: []SQLFile{},
	}

	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			catalog.Missing = true
			return catalog
		}
		catalog.Error = err.Error()
		return catalog
	}
	if !info.IsDir() {
		catalog.Error = "sql root is not a directory"
		return catalog
	}

	schemas, err := loadSQLDir(root, "schemas")
	if err != nil {
		catalog.Error = err.Error()
		return catalog
	}
	queries, err := loadSQLDir(root, "queries")
	if err != nil {
		catalog.Error = err.Error()
		return catalog
	}
	catalog.Schemas = schemas
	catalog.Queries = queries
	return catalog
}

func loadSQLDir(root, subdir string) ([]SQLFile, error) {
	dir := filepath.Join(root, subdir)
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []SQLFile{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New(subdir + " is not a directory")
	}

	files := []SQLFile{}
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".sql") {
			return nil
		}
		item, err := loadSQLFile(root, path)
		if err != nil {
			return err
		}
		files = append(files, item)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func loadSQLFile(root, path string) (SQLFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return SQLFile{}, err
	}

	rel := path
	if relPath, err := filepath.Rel(root, path); err == nil {
		rel = relPath
	}
	rel = filepath.ToSlash(rel)

	truncated := false
	if int64(len(content)) > defaultMaxSQLLen {
		content = content[:defaultMaxSQLLen]
		truncated = true
	}
	text := string(content)
	lines := 0
	if text != "" {
		lines = strings.Count(text, "\n") + 1
	}

	info, err := os.Stat(path)
	if err != nil {
		return SQLFile{}, err
	}

	return SQLFile{
		Name:      filepath.Base(path),
		Path:      rel,
		Bytes:     info.Size(),
		Lines:     lines,
		Truncated: truncated,
		Content:   text,
	}, nil
}
