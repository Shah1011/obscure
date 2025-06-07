// Package utils provides utility functions for the obscure application
package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// PromptPasswordConfirm prompts for a password and confirmation, returning the password if they match
func PromptPasswordConfirm(prompt string) (string, error) {
	password, err := PromptPassword(prompt)
	if err != nil {
		return "", err
	}

	confirm, err := PromptPassword("🔐 Confirm password: ")
	if err != nil {
		return "", err
	}

	if password != confirm {
		return "", fmt.Errorf("passwords do not match")
	}

	return password, nil
}

// PromptPassword prompts for a password
func PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // newline after password input
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytePassword)), nil
}

// PromptEmail prompts for an email address
func PromptEmail(prompt string) (string, error) {
	fmt.Print(prompt)
	var email string
	fmt.Scanln(&email)
	return strings.TrimSpace(email), nil
}

// PromptLine prompts for a single line of input
func PromptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func PromptUsername(prompt string) (string, error) {
	fmt.Print(prompt)
	var username string
	_, err := fmt.Scanln(&username)
	return strings.TrimSpace(username), err
}
