package auth

import (
	"context"
	"fmt"
	"time"

	"firebase.google.com/go/v4/auth"

	firebase "github.com/andrescris/firestore/lib/firebase"
)

// CreateUser crea un nuevo usuario sin contraseña
func CreateUser(ctx context.Context, request firebase.CreateUserRequest) (*firebase.UserRecord, error) {
	client := firebase.GetAuthClient()
	params := (&auth.UserToCreate{}).
		Email(request.Email).
		Disabled(request.Disabled)

	if request.DisplayName != "" {
		params = params.DisplayName(request.DisplayName)
	}
	if request.PhotoURL != "" {
		params = params.PhotoURL(request.PhotoURL)
	}

	record, err := client.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return mapUserRecord(record), nil
}

// GetUser obtiene un usuario por UID
func GetUser(ctx context.Context, uid string) (*firebase.UserRecord, error) {
	client := firebase.GetAuthClient()
	record, err := client.GetUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return mapUserRecord(record), nil
}

// GetUserByEmail obtiene un usuario por email
func GetUserByEmail(ctx context.Context, email string) (*firebase.UserRecord, error) {
	client := firebase.GetAuthClient()
	record, err := client.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return mapUserRecord(record), nil
}

// UpdateUser actualiza un usuario existente
func UpdateUser(ctx context.Context, uid string, request firebase.UpdateUserRequest) (*firebase.UserRecord, error) {
	client := firebase.GetAuthClient()
	params := (&auth.UserToUpdate{})

	if request.Email != nil {
		params = params.Email(*request.Email)
	}
	if request.DisplayName != nil {
		params = params.DisplayName(*request.DisplayName)
	}
	if request.PhotoURL != nil {
		params = params.PhotoURL(*request.PhotoURL)
	}
	if request.Disabled != nil {
		params = params.Disabled(*request.Disabled)
	}
	if request.EmailVerified != nil {
		params = params.EmailVerified(*request.EmailVerified)
	}
	if request.CustomClaims != nil {
		params = params.CustomClaims(request.CustomClaims)
	}

	record, err := client.UpdateUser(ctx, uid, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return mapUserRecord(record), nil
}

// CreateCustomToken crea un token personalizado
func CreateCustomToken(ctx context.Context, uid string, claims map[string]interface{}) (string, error) {
	client := firebase.GetAuthClient()
	token, err := client.CustomTokenWithClaims(ctx, uid, claims)
	if err != nil {
		return "", fmt.Errorf("failed to create custom token: %w", err)
	}
	return token, nil
}

// SetCustomClaims establece claims personalizados para un usuario
func SetCustomClaims(ctx context.Context, uid string, claims map[string]interface{}) error {
	client := firebase.GetAuthClient()
	if err := client.SetCustomUserClaims(ctx, uid, claims); err != nil {
		return fmt.Errorf("failed to set custom claims: %w", err)
	}
	return nil
}

// ListUsers lista usuarios con paginación
func ListUsers(ctx context.Context, maxResults int, pageToken string) ([]*firebase.UserRecord, string, error) {
	client := firebase.GetAuthClient()
	iterator := client.Users(ctx, pageToken)
	var users []*firebase.UserRecord
	count := 0
	for {
		if count >= maxResults {
			break
		}
		record, err := iterator.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			return nil, "", fmt.Errorf("failed to list users: %w", err)
		}
		users = append(users, mapUserRecord(record.UserRecord))
		count++
	}
	nextPageToken := ""
	if iterator.PageInfo().Token != "" {
		// Esta es una simplificación; la API real puede requerir una lógica más compleja para la paginación
		nextPageToken = iterator.PageInfo().Token
	}
	return users, nextPageToken, nil
}

// UserExists verifica si un usuario existe por UID
func UserExists(ctx context.Context, uid string) (bool, error) {
	_, err := GetUser(ctx, uid)
	if err != nil {
		if auth.IsUserNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListAllUsers es una función auxiliar para obtener todos los usuarios
func ListAllUsers(ctx context.Context) ([]*firebase.UserRecord, error) {
	client := firebase.GetAuthClient()
	iterator := client.Users(ctx, "")
	var users []*firebase.UserRecord
	for {
		record, err := iterator.Next()
		if err != nil {
			if err.Error() == "no more items in iterator" {
				break
			}
			return nil, fmt.Errorf("failed to list all users: %w", err)
		}
		users = append(users, mapUserRecord(record.UserRecord))
	}
	return users, nil
}

// GetUserCount obtiene el número total de usuarios
func GetUserCount(ctx context.Context) (int, error) {
	users, err := ListAllUsers(ctx)
	if err != nil {
		return 0, err
	}
	return len(users), nil
}


// mapUserRecord convierte auth.UserRecord a firebase.UserRecord
func mapUserRecord(record *auth.UserRecord) *firebase.UserRecord {
	return &firebase.UserRecord{
		UID:           record.UID,
		Email:         record.Email,
		DisplayName:   record.DisplayName,
		PhotoURL:      record.PhotoURL,
		Disabled:      record.Disabled,
		EmailVerified: record.EmailVerified,
		CustomClaims:  record.CustomClaims,
		CreationTime:  time.Unix(record.UserMetadata.CreationTimestamp, 0),
		LastLogInTime: time.Unix(record.UserMetadata.LastLogInTimestamp, 0),
	}
}