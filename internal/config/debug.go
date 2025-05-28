package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type JWTClaims struct {
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
}

// Debug function to check token storage
func DebugTokenStorage() {
	fmt.Println("🔍 Debugging token storage...")

	// Check .obscure directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ Failed to get home directory: %v\n", err)
		return
	}

	obscureDir := filepath.Join(homeDir, ".obscure")
	fmt.Printf("📁 Obscure directory: %s\n", obscureDir)

	// Check if directory exists
	if _, err := os.Stat(obscureDir); os.IsNotExist(err) {
		fmt.Println("❌ .obscure directory doesn't exist")
		return
	}

	// List all files in .obscure directory
	files, err := os.ReadDir(obscureDir)
	if err != nil {
		fmt.Printf("❌ Failed to read .obscure directory: %v\n", err)
		return
	}

	fmt.Println("📄 Files in .obscure directory:")
	for _, file := range files {
		fmt.Printf("  - %s (size: %d bytes)\n", file.Name(), getFileSize(filepath.Join(obscureDir, file.Name())))
	}

	// Check token file specifically
	tokenFile := filepath.Join(obscureDir, "token")
	if _, err := os.Stat(tokenFile); err == nil {
		fmt.Println("✅ Token file exists")

		// Read token content
		content, err := os.ReadFile(tokenFile)
		if err != nil {
			fmt.Printf("❌ Failed to read token file: %v\n", err)
			return
		}

		tokenStr := string(content)
		fmt.Printf("📋 Token length: %d characters\n", len(tokenStr))
		fmt.Printf("📋 Token starts with: %s...\n", truncateString(tokenStr, 20))
		fmt.Printf("📋 Token ends with: ...%s\n", truncateString(reverseString(tokenStr), 20))

		// Check for common issues
		if strings.HasPrefix(tokenStr, "\n") || strings.HasSuffix(tokenStr, "\n") {
			fmt.Println("⚠️  Token has newlines (this could cause issues)")
		}

		if strings.Contains(tokenStr, " ") {
			fmt.Println("⚠️  Token contains spaces (this could cause issues)")
		}

		// Try to validate token format (basic JWT check)
		parts := strings.Split(strings.TrimSpace(tokenStr), ".")
		if len(parts) == 3 {
			fmt.Println("✅ Token appears to be in JWT format")
		} else {
			fmt.Printf("⚠️  Token doesn't appear to be JWT format (has %d parts, expected 3)\n", len(parts))
		}

	} else {
		fmt.Println("❌ Token file doesn't exist")
	}
}

// GetSessionTokenWithDebug returns token with debug info
func GetSessionTokenWithDebug() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	tokenFile := filepath.Join(homeDir, ".obscure", "token")

	content, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("failed to read token file: %w", err)
	}

	// Clean the token (remove any whitespace)
	token := strings.TrimSpace(string(content))

	fmt.Printf("🔍 Raw token length: %d\n", len(string(content)))
	fmt.Printf("🔍 Cleaned token length: %d\n", len(token))

	return token, nil
}

// Helper functions
func getFileSize(path string) int64 {
	if info, err := os.Stat(path); err == nil {
		return info.Size()
	}
	return 0
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func DebugUsersJSON() {
	fmt.Println("🔍 Debugging users.json structure...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ Failed to get home directory: %v\n", err)
		return
	}

	usersFile := filepath.Join(homeDir, ".obscure", "users.json")

	content, err := os.ReadFile(usersFile)
	if err != nil {
		fmt.Printf("❌ Failed to read users.json: %v\n", err)
		return
	}

	// Parse JSON structure
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		fmt.Printf("❌ Failed to parse JSON: %v\n", err)
		return
	}

	fmt.Println("📋 users.json structure:")
	for email, value := range data {
		fmt.Printf("👤 User: %s\n", email)

		if userObj, ok := value.(map[string]interface{}); ok {
			for key, val := range userObj {
				switch v := val.(type) {
				case string:
					if key == "token" || strings.Contains(strings.ToLower(key), "token") {
						fmt.Printf("    🔑 %s: [TOKEN - length: %d]\n", key, len(v))

						// Check if it's a JWT token
						if strings.Count(v, ".") == 2 {
							fmt.Printf("      ✅ Appears to be JWT format\n")

							// Try to decode and check expiry
							parts := strings.Split(v, ".")
							if len(parts) == 3 {
								// Decode payload
								payload := parts[1]
								for len(payload)%4 != 0 {
									payload += "="
								}

								decoded, err := base64.URLEncoding.DecodeString(payload)
								if err == nil {
									var claims JWTClaims
									if err := json.Unmarshal(decoded, &claims); err == nil {
										issuedAt := time.Unix(claims.Iat, 0)
										expiresAt := time.Unix(claims.Exp, 0)
										now := time.Now()

										fmt.Printf("      📅 Issued at: %s\n", issuedAt.Format("2006-01-02 15:04:05"))
										fmt.Printf("      📅 Expires at: %s\n", expiresAt.Format("2006-01-02 15:04:05"))
										fmt.Printf("      📅 Current time: %s\n", now.Format("2006-01-02 15:04:05"))

										if now.After(expiresAt) {
											fmt.Printf("      ❌ TOKEN IS EXPIRED (expired %v ago)\n", now.Sub(expiresAt).Round(time.Minute))
										} else {
											fmt.Printf("      ✅ Token is still valid (expires in %v)\n", expiresAt.Sub(now).Round(time.Minute))
										}
									}
								}
							}
						} else {
							fmt.Printf("      ⚠️  Not in JWT format\n")
						}
					} else {
						fmt.Printf("    📝 %s: %s\n", key, v)
					}
				default:
					fmt.Printf("    📄 %s: %v\n", key, val)
				}
			}
		}
		fmt.Println() // Add empty line between users
	}

	// Show current session info
	fmt.Println("🎯 Current Session:")
	if currentEmail, err := GetSessionEmail(); err == nil {
		fmt.Printf("   📧 Email: %s\n", currentEmail)
		if currentUser, ok := data[currentEmail].(map[string]interface{}); ok {
			if token, ok := currentUser["token"].(string); ok {
				fmt.Printf("   🔑 Token length: %d\n", len(token))

				// Check current user's token expiry
				if strings.Count(token, ".") == 2 {
					parts := strings.Split(token, ".")
					if len(parts) == 3 {
						payload := parts[1]
						for len(payload)%4 != 0 {
							payload += "="
						}

						if decoded, err := base64.URLEncoding.DecodeString(payload); err == nil {
							var claims JWTClaims
							if err := json.Unmarshal(decoded, &claims); err == nil {
								expiresAt := time.Unix(claims.Exp, 0)
								now := time.Now()

								if now.After(expiresAt) {
									fmt.Printf("   ❌ CURRENT TOKEN IS EXPIRED (expired %v ago)\n", now.Sub(expiresAt).Round(time.Minute))
								} else {
									fmt.Printf("   ✅ Current token is valid (expires in %v)\n", expiresAt.Sub(now).Round(time.Minute))
								}
							}
						}
					}
				}
			}
		}
	}
}

// Add this to your debug.go file

func DebugTokenDetails() {
	fmt.Println("🔍 Detailed Token Analysis...")

	// Get the current session token
	token, err := GetSessionToken()
	if err != nil {
		fmt.Printf("❌ Failed to get session token: %v\n", err)
		return
	}

	fmt.Printf("📋 Token length: %d characters\n", len(token))
	fmt.Printf("📋 Token starts with: %s...\n", truncateString(token, 30))
	fmt.Printf("📋 Token ends with: ...%s\n", token[len(token)-30:])

	// Validate JWT structure
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		fmt.Printf("❌ Invalid JWT structure: has %d parts instead of 3\n", len(parts))
		return
	}

	fmt.Println("✅ Valid JWT structure (3 parts)")

	// Decode header
	header, err := base64.URLEncoding.DecodeString(addPadding(parts[0]))
	if err != nil {
		fmt.Printf("❌ Failed to decode JWT header: %v\n", err)
	} else {
		fmt.Printf("📋 JWT Header: %s\n", string(header))
	}

	// Decode payload
	payload, err := base64.URLEncoding.DecodeString(addPadding(parts[1]))
	if err != nil {
		fmt.Printf("❌ Failed to decode JWT payload: %v\n", err)
		return
	}

	fmt.Printf("📋 JWT Payload (first 200 chars): %s...\n", truncateString(string(payload), 200))

	// Parse claims
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		fmt.Printf("❌ Failed to parse JWT claims: %v\n", err)
		return
	}

	// Check critical Firebase claims
	fmt.Println("🔍 Firebase Token Claims:")

	if iss, ok := claims["iss"].(string); ok {
		fmt.Printf("📋 Issuer (iss): %s\n", iss)
		if !strings.Contains(iss, "firebase") && !strings.Contains(iss, "securetoken.google.com") {
			fmt.Printf("⚠️  WARNING: Issuer doesn't look like Firebase!\n")
		}
	} else {
		fmt.Printf("❌ Missing or invalid 'iss' claim\n")
	}

	if aud, ok := claims["aud"].(string); ok {
		fmt.Printf("📋 Audience (aud): %s\n", aud)
	} else {
		fmt.Printf("❌ Missing or invalid 'aud' claim\n")
	}

	if sub, ok := claims["sub"].(string); ok {
		fmt.Printf("📋 Subject (sub): %s\n", sub)
	} else {
		fmt.Printf("❌ Missing or invalid 'sub' claim\n")
	}

	if exp, ok := claims["exp"].(float64); ok {
		expiresAt := time.Unix(int64(exp), 0)
		now := time.Now()
		fmt.Printf("📅 Expires at: %s\n", expiresAt.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("📅 Current time: %s\n", now.Format("2006-01-02 15:04:05 MST"))

		if now.After(expiresAt) {
			fmt.Printf("❌ TOKEN IS EXPIRED (expired %v ago)\n", now.Sub(expiresAt).Round(time.Minute))
		} else {
			fmt.Printf("✅ Token is valid (expires in %v)\n", expiresAt.Sub(now).Round(time.Minute))
		}
	} else {
		fmt.Printf("❌ Missing or invalid 'exp' claim\n")
	}

	if iat, ok := claims["iat"].(float64); ok {
		issuedAt := time.Unix(int64(iat), 0)
		fmt.Printf("📅 Issued at: %s\n", issuedAt.Format("2006-01-02 15:04:05 MST"))
	}

	// Check auth_time for Firebase
	if authTime, ok := claims["auth_time"].(float64); ok {
		authenticatedAt := time.Unix(int64(authTime), 0)
		fmt.Printf("📅 Authenticated at: %s\n", authenticatedAt.Format("2006-01-02 15:04:05 MST"))
	}

	// Check Firebase-specific claims
	if firebase, ok := claims["firebase"].(map[string]interface{}); ok {
		fmt.Println("🔥 Firebase-specific claims found:")
		for key, value := range firebase {
			fmt.Printf("   %s: %v\n", key, value)
		}
	} else {
		fmt.Printf("⚠️  No Firebase-specific claims found\n")
	}

	// Show email from token
	if email, ok := claims["email"].(string); ok {
		fmt.Printf("📧 Email in token: %s\n", email)
	}

	if emailVerified, ok := claims["email_verified"].(bool); ok {
		fmt.Printf("✅ Email verified: %t\n", emailVerified)
	}
}

// Helper function to add base64 padding
func addPadding(s string) string {
	for len(s)%4 != 0 {
		s += "="
	}
	return s
}

// Add this to your existing debug command
func DebugBackendCommunication() {
	fmt.Println("🔍 Testing Backend Communication...")

	token, err := GetSessionToken()
	if err != nil {
		fmt.Printf("❌ Failed to get token: %v\n", err)
		return
	}

	// Show exactly what headers we're sending
	fmt.Println("📡 Request headers that will be sent:")
	fmt.Printf("   Authorization: Bearer %s...\n", truncateString(token, 20))
	fmt.Printf("   Content-Type: application/octet-stream\n")

	// Test a simple API call to see what the backend responds with
	fmt.Println("🧪 Testing simple API call...")

	// You might want to add a simple health check endpoint to your backend
	// Or test with a minimal request to see the exact error response
}
