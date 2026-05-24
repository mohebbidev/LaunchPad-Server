package storage


import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Storage struct {}

func (strg *Storage) Save(path string, r io.Reader) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (strg *Storage) Unzip(zipFile, dst string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		// prevent zip slip
		name := filepath.Clean(f.Name)
		if strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
			return fmt.Errorf("invalid file path in zip: %s", f.Name)
		}
		path := filepath.Join(dst, name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		_, err = io.Copy(out, rc)
		closeErr := out.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}


