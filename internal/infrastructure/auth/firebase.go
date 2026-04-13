package auth

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
)

// FirebaseAuth wraps the Firebase Admin SDK Auth client for token verification.
type FirebaseAuth struct {
	client *firebaseAuth.Client
}

// NewFirebaseAuth initializes the Firebase Admin SDK using the standard
// GOOGLE_APPLICATION_CREDENTIALS environment variable.
func NewFirebaseAuth() *FirebaseAuth {
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("failed to initialize Firebase app: %v", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("failed to initialize Firebase Auth client: %v", err)
	}

	log.Println("Firebase Auth initialized successfully")
	return &FirebaseAuth{client: client}
}

// VerifyToken verifies a Firebase ID token and returns the user's UID.
func (f *FirebaseAuth) VerifyToken(tokenString string) (string, error) {
	ctx := context.Background()
	token, err := f.client.VerifyIDToken(ctx, tokenString)
	if err != nil {
		return "", fmt.Errorf("invalid Firebase token: %w", err)
	}
	return token.UID, nil
}
