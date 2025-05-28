package firebase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

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

func InitApp() *firebase.App {
	once.Do(func() {
		opt := option.WithCredentialsFile("serviceAccountKey.json")
		config := &firebase.Config{
			ProjectID: "obscure-a45f2",
		}

		var err error
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
