package utils

import (
	"fmt"
	"os"
	"strings"
)

type ProgressReader struct {
	file      *os.File
	totalSize int64
	readSoFar int64
	barWidth  int
	label     string
}

func NewProgressReader(file *os.File, totalSize int64, barWidth int, label string) *ProgressReader {
	return &ProgressReader{
		file:      file,
		totalSize: totalSize,
		readSoFar: 0,
		barWidth:  barWidth,
		label:     label,
	}
}

func (r *ProgressReader) Read(p []byte) (int, error) {
	n, err := r.file.Read(p)
	if n > 0 {
		r.readSoFar += int64(n)
		DrawProgressBar(r.readSoFar, r.totalSize, r.barWidth, r.label)
	}
	return n, err
}

// DrawProgressBar prints a horizontal progress bar to the terminal.
func DrawProgressBar(current, total int64, barWidth int, label string) {
	percent := float64(current) / float64(total)
	filled := int(percent * float64(barWidth))
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	fmt.Printf("\r🔹 %s [%s] %3.0f%%", label, bar, percent*100)
	if current == total {
		fmt.Println()
	}
}
