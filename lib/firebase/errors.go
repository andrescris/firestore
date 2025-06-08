package firebase

import "fmt"

// Errores globales
var (
	ErrMissingCredentials = &MissingCredentialsError{}
	ErrDocumentNotFound   = &DocumentNotFoundError{}
	ErrFileNotFound       = &FileNotFoundError{}
	ErrUserNotFound       = &UserNotFoundError{}
)

// MissingCredentialsError cuando no se encuentran las credenciales
type MissingCredentialsError struct{}

func (e *MissingCredentialsError) Error() string {
	return "FIREBASE_SERVICE_ACCOUNT environment variable not set"
}

// DocumentNotFoundError cuando no se encuentra un documento en Firestore
type DocumentNotFoundError struct {
	Collection string
	DocumentID string
}

func (e *DocumentNotFoundError) Error() string {
	return fmt.Sprintf("document not found in collection '%s' with ID '%s'", e.Collection, e.DocumentID)
}

// FileNotFoundError cuando no se encuentra un archivo en Storage
type FileNotFoundError struct {
	Bucket   string
	FileName string
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("file not found in bucket '%s' with name '%s'", e.Bucket, e.FileName)
}

// UserNotFoundError cuando no se encuentra un usuario en Auth
type UserNotFoundError struct {
	Identifier string // Puede ser UID, email, etc.
}

func (e *UserNotFoundError) Error() string {
	return fmt.Sprintf("user not found: %s", e.Identifier)
}