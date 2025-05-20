package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func ZipDirectoryToBuffer(srcDir string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	err := filepath.Walk(srcDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		fileInZip, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		fileOnDisk, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fileOnDisk.Close()

		_, err = io.Copy(fileInZip, fileOnDisk)
		return err
	})

	if err != nil {
		return nil, err
	}

	err = zipWriter.Close()
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// UnzipFromStream extracts files from a ZIP archive stream into a target directory.
func UnzipFromStream(zipReader io.Reader, outputDir string) error {
	// Read all data into memory (as zip.NewReader requires a Seeker)
	data, err := io.ReadAll(zipReader)
	if err != nil {
		return fmt.Errorf("failed to read zip stream: %w", err)
	}

	// Create a zip.Reader from the in-memory byte slice
	zipReaderAt := bytes.NewReader(data)
	zipReaderObj, err := zip.NewReader(zipReaderAt, int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Iterate over files in the ZIP archive
	for _, f := range zipReaderObj.File {
		filePath := filepath.Join(outputDir, f.Name)

		// Ensure the path is within the outputDir (prevent ZipSlip)
		if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(outputDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", filePath)
		}

		// Handle directories
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filePath, err)
			}
			continue
		}

		// Create parent directories if necessary
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directories: %w", err)
		}

		// Open file in the archive
		srcFile, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", f.Name, err)
		}
		defer srcFile.Close()

		// Create destination file
		destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}

		// Copy file contents
		if _, err := io.Copy(destFile, srcFile); err != nil {
			destFile.Close()
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
		destFile.Close()
	}

	return nil
}
