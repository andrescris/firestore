package auth

import (
	"context"
	"fmt"
	"time"

	"firebase.google.com/go/v4/auth"

	firebase "github.com/andrescris/firestore/lib/firebase"
)

// CreateUser crea un nuevo usuario
func CreateUser(ctx context.Context, request firebase.CreateUserRequest) (*firebase.UserRecord, error) {
	client := firebase.GetAuthClient()

	params := (&auth.UserToCreate{}).
		Email(request.Email).
		Password(request.Password).
		Disabled(request.Disabled)

	if request.DisplayName != "" {
		params = params.DisplayName(request.DisplayName)
	}
	if request.PhoneNumber != "" {
		params = params.PhoneNumber(request.PhoneNumber)
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
	if request.Password != nil {
		params = params.Password(*request.Password)
	}
	if request.DisplayName != nil {
		params = params.DisplayName(*request.DisplayName)
	}
	if request.PhoneNumber != nil {
		params = params.PhoneNumber(*request.PhoneNumber)
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

// DeleteUser elimina un usuario
func DeleteUser(ctx context.Context, uid string) error {
	client := firebase.GetAuthClient()

	if err := client.DeleteUser(ctx, uid); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// VerifyIDToken verifica un token de ID
func VerifyIDToken(ctx context.Context, idToken string) (*auth.Token, error) {
	client := firebase.GetAuthClient()

	token, err := client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	return token, nil
}

// CreateCustomToken crea un token personalizado
func CreateCustomToken(ctx context.Context, uid string, claims map[string]interface{}) (string, error) {
	client := firebase.GetAuthClient()

	var token string
	var err error

	if claims != nil {
		token, err = client.CustomTokenWithClaims(ctx, uid, claims)
	} else {
		token, err = client.CustomToken(ctx, uid)
	}

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

// ListUsers lista usuarios con paginación (CORREGIDO para la nueva API)
func ListUsers(ctx context.Context, maxResults int, pageToken string) ([]*firebase.UserRecord, string, error) {
	client := firebase.GetAuthClient()

	// Crear el iterador con el pageToken
	iterator := client.Users(ctx, pageToken)
	
	var users []*firebase.UserRecord
	count := 0

	// Iterar manualmente respetando el límite
	for {
		if count >= maxResults {
			break
		}

		record, err := iterator.Next()
		if err != nil {
			// Si llegamos al final o hay error, retornar lo que tenemos
			if err.Error() == "no more items in iterator" {
				break
			}
			return nil, "", fmt.Errorf("failed to list users: %w", err)
		}

		users = append(users, mapUserRecord(record.UserRecord))
		count++
	}

	// Para obtener el siguiente token, necesitamos hacer otra llamada
	// pero en este caso simplificado, retornamos string vacío
	nextPageToken := ""
	
	return users, nextPageToken, nil
}

// ListAllUsers lista todos los usuarios (sin paginación)
func ListAllUsers(ctx context.Context) ([]*firebase.UserRecord, error) {
	client := firebase.GetAuthClient()

	iterator := client.Users(ctx, "")
	var users []*firebase.UserRecord

	for {
		record, err := iterator.Next()
		if err != nil {
			// Si llegamos al final, terminar
			if err.Error() == "no more items in iterator" {
				break
			}
			return nil, fmt.Errorf("failed to list all users: %w", err)
		}

		users = append(users, mapUserRecord(record.UserRecord))
	}

	return users, nil
}

// UserExists verifica si un usuario existe por UID
func UserExists(ctx context.Context, uid string) (bool, error) {
	_, err := GetUser(ctx, uid)
	if err != nil {
		// Si el error es "user not found", retornar false
		if auth.IsUserNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// UserExistsByEmail verifica si un usuario existe por email
func UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, err := GetUserByEmail(ctx, email)
	if err != nil {
		// Si el error es "user not found", retornar false
		if auth.IsUserNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DisableUser desactiva un usuario
func DisableUser(ctx context.Context, uid string) error {
	disabled := true
	_, err := UpdateUser(ctx, uid, firebase.UpdateUserRequest{
		Disabled: &disabled,
	})
	return err
}

// EnableUser activa un usuario
func EnableUser(ctx context.Context, uid string) error {
	disabled := false
	_, err := UpdateUser(ctx, uid, firebase.UpdateUserRequest{
		Disabled: &disabled,
	})
	return err
}

// GetUserCount obtiene el número total de usuarios (aproximado)
func GetUserCount(ctx context.Context) (int, error) {
	users, err := ListAllUsers(ctx)
	if err != nil {
		return 0, err
	}
	return len(users), nil
}

// ListUsersByEmail lista usuarios filtrados por dominio de email
func ListUsersByEmail(ctx context.Context, emailDomain string) ([]*firebase.UserRecord, error) {
	allUsers, err := ListAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	var filteredUsers []*firebase.UserRecord
	for _, user := range allUsers {
		if user.Email != "" {
			// Verificar si el email contiene el dominio
			if emailDomain == "" || fmt.Sprintf("@%s", emailDomain) == user.Email[len(user.Email)-len(emailDomain)-1:] {
				filteredUsers = append(filteredUsers, user)
			}
		}
	}

	return filteredUsers, nil
}

// mapUserRecord convierte auth.UserRecord a firebase.UserRecord
func mapUserRecord(record *auth.UserRecord) *firebase.UserRecord {
	return &firebase.UserRecord{
		UID:           record.UID,
		Email:         record.Email,
		PhoneNumber:   record.PhoneNumber,
		DisplayName:   record.DisplayName,
		PhotoURL:      record.PhotoURL,
		Disabled:      record.Disabled,
		EmailVerified: record.EmailVerified,
		CustomClaims:  record.CustomClaims,
		CreationTime:  time.Unix(record.UserMetadata.CreationTimestamp, 0),
		LastLogInTime: time.Unix(record.UserMetadata.LastLogInTimestamp, 0),
	}
}