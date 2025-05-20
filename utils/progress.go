package utils

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"
)

type ProgressBuffer struct {
	reader       *bytes.Reader // now a ReadSeeker
	description  string
	totalBytes   int64
	readBytes    int64
	barWidth     int
	lastProgress int
}

func NewProgressBuffer(data []byte, description string, barWidth int) *ProgressBuffer {
	return &ProgressBuffer{
		reader:      bytes.NewReader(data),
		description: description,
		totalBytes:  int64(len(data)),
		barWidth:    barWidth,
	}
}

func (pb *ProgressBuffer) Read(p []byte) (int, error) {
	n, err := pb.reader.Read(p)
	pb.readBytes += int64(n)

	progress := int((float64(pb.readBytes) / float64(pb.totalBytes)) * float64(pb.barWidth))
	if progress != pb.lastProgress {
		fmt.Printf("\r%s [%s%s] %d%%",
			pb.description,
			strings.Repeat("#", progress),
			strings.Repeat(" ", pb.barWidth-progress),
			int(float64(pb.readBytes)*100/float64(pb.totalBytes)),
		)
		pb.lastProgress = progress
	}

	if err == io.EOF {
		fmt.Println()
	}

	return n, err
}

func (pb *ProgressBuffer) Seek(offset int64, whence int) (int64, error) {
	pos, err := pb.reader.Seek(offset, whence)
	if err == nil {
		pb.readBytes = pos // Keep progress in sync
		pb.lastProgress = int((float64(pos) / float64(pb.totalBytes)) * float64(pb.barWidth))
	}
	return pos, err
}

type ProgressWriter struct {
	Writer       io.Writer
	Description  string
	BarWidth     int
	TotalBytes   int64
	writtenBytes int64
	lastPrint    time.Time
}

func NewProgressWriter(w io.Writer, description string, barWidth int, totalBytes int64) *ProgressWriter {
	return &ProgressWriter{
		Writer:      w,
		Description: description,
		BarWidth:    barWidth,
		TotalBytes:  totalBytes,
		lastPrint:   time.Now(),
	}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	pw.writtenBytes += int64(n)

	if time.Since(pw.lastPrint) > 200*time.Millisecond || err == io.EOF {
		pw.printProgress()
		pw.lastPrint = time.Now()
	}

	return n, err
}

func (pw *ProgressWriter) printProgress() {
	if pw.TotalBytes > 0 {
		percent := float64(pw.writtenBytes) / float64(pw.TotalBytes)
		progress := int(percent * float64(pw.BarWidth))
		fmt.Printf("\r%s [%s%s] %3.0f%%",
			pw.Description,
			strings.Repeat("#", progress),
			strings.Repeat(" ", pw.BarWidth-progress),
			percent*100,
		)
		if pw.writtenBytes >= pw.TotalBytes {
			fmt.Println()
		}
	} else {
		fmt.Printf("\r%s: %d KB", pw.Description, pw.writtenBytes/1024)
	}
}
