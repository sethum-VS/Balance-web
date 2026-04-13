package templates

// FirebaseConfig holds the Firebase Web SDK configuration values
// passed from server-side environment variables to the login template.
type FirebaseConfig struct {
	APIKey            string
	AuthDomain        string
	ProjectID         string
	StorageBucket     string
	MessagingSenderID string
	AppID             string
}
