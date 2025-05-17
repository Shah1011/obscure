package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type ProgressReader struct {
	Reader       io.Reader
	Total        int64
	ReadBytes    int64
	LastReported int64
}

func NewProgressReader(r io.Reader, total int64) *ProgressReader {
	return &ProgressReader{
		Reader: r,
		Total:  total,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.ReadBytes += int64(n)

	if pr.Total > 0 {
		progress := pr.ReadBytes * 100 / pr.Total
		if progress > 100 {
			progress = 100
		}

		currentBucket := (progress / 10) * 10

		if currentBucket > pr.LastReported {
			pr.LastReported = currentBucket
			// Pass progress (0-100) to your DrawProgressBar
			DrawProgressBar(progress, 100, 30) // or however your function signature looks
		}
	}

	return n, err
}

func ZipDirectory(sourceDir string, targetZip string) error {
	zipFile, err := os.Create(targetZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	fmt.Println("🔹 Starting to zip directory:", sourceDir)

	// Get absolute path of the zip file to exclude it
	absTargetZip, err := filepath.Abs(targetZip)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of target zip: %v", err)
	}

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the zip file itself
		absCurrentPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if absCurrentPath == absTargetZip {
			return nil // ❌ skip the zip file
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		if info.IsDir() {
			fmt.Printf("📁 Adding directory: %s/\n", relPath)
		} else {
			fmt.Printf("📄 Adding file: %s\n", relPath)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = relPath
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			fileInfo, err := file.Stat()
			if err != nil {
				return err
			}

			progressReader := NewProgressReader(file, fileInfo.Size())
			_, err = io.Copy(writer, progressReader)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println("✅ Zipping complete")
	return nil
}
