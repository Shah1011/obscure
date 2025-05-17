package utils

import (
	"fmt"
	"strings"
)

// DrawProgressBar prints a horizontal progress bar to the terminal.
func DrawProgressBar(current, total int64, barWidth int) {
	percent := float64(current) / float64(total)
	filled := int(percent * float64(barWidth))
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	fmt.Printf("\r🔹 Compression progress: [%s] %3.0f%%", bar, percent*100)
	if current == total {
		fmt.Println() // Move to the next line when done
	}
}
