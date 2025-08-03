package firebase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	app  *firebase.App
	once sync.Once
)

// FirebaseCredentials holds both service account and API key
type FirebaseCredentials struct {
	ServiceAccount   json.RawMessage `json:"serviceAccount"`
	FirebaseApiKey   string          `json:"firebaseApiKey"`
}

// Global variable to cache Firebase API key
var cachedFirebaseApiKey string

// fetchFirebaseCredentialsFromVercel fetches both service account and API key from Vercel API
func fetchFirebaseCredentialsFromVercel() (*FirebaseCredentials, error) {
	endpoint := os.Getenv("VERCEL_ENDPOINT")
	apiKey := os.Getenv("VERCEL_API_KEY")
	
	if endpoint == "" || apiKey == "" {
		return nil, fmt.Errorf("VERCEL_ENDPOINT or VERCEL_API_KEY not set")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Vercel API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Vercel API returned status %d", resp.StatusCode)
	}

	// Read and parse response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var credentials FirebaseCredentials
	if err := json.Unmarshal(body, &credentials); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Cache the Firebase API key globally
	cachedFirebaseApiKey = credentials.FirebaseApiKey

	return &credentials, nil
}

func InitApp() *firebase.App {
	once.Do(func() {
		var opt option.ClientOption
		var err error

		// Try to fetch credentials from Vercel API first
		credentials, err := fetchFirebaseCredentialsFromVercel()
		if err != nil {
			log.Printf("Failed to fetch from Vercel API: %v", err)
			log.Printf("Falling back to local serviceAccountKey.json file...")
			
			// Fallback to local file
			if _, fileErr := os.Stat("serviceAccountKey.json"); fileErr == nil {
				opt = option.WithCredentialsFile("serviceAccountKey.json")
				// Also try to get Firebase API key from environment as fallback
				if envApiKey := os.Getenv("FIREBASE_API_KEY"); envApiKey != "" {
					cachedFirebaseApiKey = envApiKey
				}
			} else {
				log.Fatalf("No Firebase credentials available. Vercel API failed and local file not found.")
			}
		} else {
			// Use credentials from Vercel API
			opt = option.WithCredentialsJSON(credentials.ServiceAccount)
			log.Printf("Successfully fetched Firebase credentials from Vercel API")
		}

		config := &firebase.Config{
			ProjectID: "obscure-a45f2",
		}

		app, err = firebase.NewApp(context.Background(), config, opt)
		if err != nil {
			log.Fatalf("Error initializing Firebase app: %v", err)
		}
	})
	return app
}

func GetFirestoreClient() (*firestore.Client, error) {
	app := InitApp()
	return app.Firestore(context.Background())
}

func SignUpUser(email, password string) (*auth.UserRecord, error) {
	app := InitApp()
	client, err := app.Auth(context.Background())
	if err != nil {
		return nil, err
	}

	params := (&auth.UserToCreate{}).
		Email(email).
		Password(password)

	return client.CreateUser(context.Background(), params)
}

// GetFirebaseApiKey returns the Firebase API key (from Vercel API or environment)
func GetFirebaseApiKey() string {
	// If we have a cached API key from Vercel, use it
	if cachedFirebaseApiKey != "" {
		return cachedFirebaseApiKey
	}
	
	// Fallback to environment variable
	return os.Getenv("FIREBASE_API_KEY")
}

func FirebaseLogin(email, password, apiKey string) (string, error) {
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", apiKey)

	data := map[string]interface{}{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	}
	body, _ := json.Marshal(data)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("invalid email or password")
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	return result["idToken"].(string), nil
}

func SaveUserData(uid, username, provider string) error {
	app := InitApp()
	client, err := app.Firestore(context.Background())
	if err != nil {
		return err
	}
	defer client.Close()

	// Get user email from Auth
	authClient, err := app.Auth(context.Background())
	if err != nil {
		return err
	}
	user, err := authClient.GetUser(context.Background(), uid)
	if err != nil {
		return err
	}

	_, err = client.Collection("users").Doc(uid).Set(context.Background(), map[string]interface{}{
		"username":        username,
		"defaultProvider": provider,
		"email":           user.Email,
	})
	return err
}

// Check if a user already exists with the email
func UserEmailExists(email string) (bool, error) {
	client, err := GetFirestoreClient()
	if err != nil {
		return false, err
	}
	defer client.Close()

	iter := client.Collection("users").Where("email", "==", email).Documents(context.Background())
	doc, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return doc.Exists(), nil
}

// Check if username is already taken
func UsernameTaken(username string) (bool, error) {
	client, err := GetFirestoreClient()
	if err != nil {
		return false, err
	}
	defer client.Close()

	iter := client.Collection("users").Where("username", "==", username).Documents(context.Background())
	doc, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return doc.Exists(), nil
}
