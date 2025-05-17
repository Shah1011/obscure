package utils

import (
	"fmt"
	"syscall"

	"golang.org/x/term"
)

// Securely prompts the user for a password without echoing it
func PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(passwordBytes), nil
}
