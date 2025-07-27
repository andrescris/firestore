package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"

	firebase "github.com/andrescris/firestore/lib/firebase"
	"github.com/andrescris/firestore/lib/firebase/firestore"
)

// LoginResponse respuesta del login
type LoginResponse struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	User         *firebase.UserRecord   `json:"user,omitempty"`
	CustomToken  string                 `json:"custom_token,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	ExpiresAt    time.Time              `json:"expires_at,omitempty"`
	Claims       map[string]interface{} `json:"claims,omitempty"`
}

// RequestOTPResponse respuesta de la solicitud de OTP
type RequestOTPResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RequestOTP genera y envía un OTP al usuario
func RequestOTP(ctx context.Context, request firebase.RequestOTPRequest) (*RequestOTPResponse, error) {
	// 1. Validar que el usuario existe
	user, err := GetUserByEmail(ctx, request.Email)
	if err != nil {
		return &RequestOTPResponse{Success: false, Message: "Usuario no encontrado."}, nil
	}

	// 2. Generar un OTP numérico de 6 dígitos
	otp, err := generateNumericOTP(6)
	if err != nil {
		return nil, fmt.Errorf("error generating OTP: %w", err)
	}

	// 3. Guardar el OTP en Firestore
	expiresAt := time.Now().Add(10 * time.Minute) // OTP válido por 10 minutos
	otpData := map[string]interface{}{
		"uid":         user.UID,
		"email":       user.Email,
		"otp":         otp,
		"expires_at":  expiresAt,
		"used":        false,
	}

	_, err = firestore.CreateDocument(ctx, "user_otps", otpData)
	if err != nil {
		return nil, fmt.Errorf("error saving OTP: %w", err)
	}

	// 4. Enviar el OTP (simulado)
	log.Printf("✅ OTP para %s: %s (Válido por 10 minutos)", user.Email, otp)

	return &RequestOTPResponse{
		Success: true,
		Message: "Se ha enviado un código de un solo uso a tu correo.",
	}, nil
}

// LoginWithOTP autentica a un usuario usando un OTP
func LoginWithOTP(ctx context.Context, request firebase.LoginWithOTPRequest) (*LoginResponse, error) {
	// 1. Buscar el OTP en Firestore
	queryOptions := firebase.QueryOptions{
		Filters: []firebase.QueryFilter{
			{Field: "email", Operator: "==", Value: request.Email},
			{Field: "otp", Operator: "==", Value: request.OTP},
			{Field: "used", Operator: "==", Value: false},
		},
		OrderBy:  "created_at",
		OrderDir: "desc",
		Limit:    1,
	}

	otpDocs, err := firestore.QueryDocuments(ctx, "user_otps", queryOptions)
	if err != nil || len(otpDocs) == 0 {
		return &LoginResponse{Success: false, Message: "OTP inválido o no encontrado."}, nil
	}
	
	otpDoc := otpDocs[0]

	// 2. Verificar si el OTP ha expirado
	expiresAt := otpDoc.Data["expires_at"].(time.Time)
	if time.Now().After(expiresAt) {
		return &LoginResponse{Success: false, Message: "El OTP ha expirado."}, nil
	}

	// 3. Marcar el OTP como usado
	err = firestore.UpdateDocument(ctx, "user_otps", otpDoc.ID, map[string]interface{}{"used": true})
	if err != nil {
		return nil, fmt.Errorf("error marking OTP as used: %w", err)
	}
	
	// 4. Obtener datos del usuario
	uid := otpDoc.Data["uid"].(string)
	user, err := GetUser(ctx, uid)
	if err != nil {
		return &LoginResponse{Success: false, Message: "No se pudo verificar al usuario."}, nil
	}
	
	// 5. Crear token personalizado y sesión
	claims, _ := getUserClaims(ctx, uid)
	customToken, err := CreateCustomToken(ctx, user.UID, claims)
	if err != nil {
		return nil, fmt.Errorf("error creating custom token: %w", err)
	}
	
	sessionID, sessionExpiresAt, err := createSession(ctx, user.UID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}

	// 6. Actualizar último login
	updateLastLogin(ctx, user.UID)

	return &LoginResponse{
		Success:     true,
		Message:     "Login exitoso",
		User:        user,
		CustomToken: customToken,
		SessionID:   sessionID,
		ExpiresAt:   sessionExpiresAt,
		Claims:      claims,
	}, nil
}

type SessionInfo struct {
	UID       string
	Email     string
	Active    bool
	Claims    map[string]interface{}
	ExpiresAt time.Time
}

// ValidateSession verifica si una sesión es válida y activa.
func ValidateSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	doc, err := firestore.GetDocument(ctx, "user_sessions", sessionID)
	if err != nil {
		return nil, fmt.Errorf("sesión no encontrada")
	}

	active, _ := doc.Data["active"].(bool)
	expiresAt, _ := doc.Data["expires_at"].(time.Time)

	if !active || time.Now().After(expiresAt) {
		return nil, fmt.Errorf("sesión inactiva o expirada")
	}

	uid, _ := doc.Data["uid"].(string)
	email, _ := doc.Data["email"].(string)
	claims, err := getUserClaims(ctx, uid)
	if err != nil {
		claims = make(map[string]interface{}) // Si no hay claims, devolver mapa vacío
	}

	return &SessionInfo{
		UID:       uid,
		Email:     email,
		Active:    true,
		Claims:    claims,
		ExpiresAt: expiresAt,
	}, nil
}

// Logout invalida una sesión de usuario.
func Logout(ctx context.Context, sessionID string) error {
	return firestore.UpdateDocument(ctx, "user_sessions", sessionID, map[string]interface{}{
		"active": false,
	})
}

// --- FUNCIONES AUXILIARES ---

func generateNumericOTP(length int) (string, error) {
	const digits = "0123456789"
	otp := make([]byte, length)
	for i := range otp {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[num.Int64()]
	}
	return string(otp), nil
}

func getUserClaims(ctx context.Context, uid string) (map[string]interface{}, error) {
	doc, err := firestore.GetDocument(ctx, "user_claims", uid)
	if err != nil {
		return nil, err
	}
	return doc.Data["claims"].(map[string]interface{}), nil
}

func createSession(ctx context.Context, uid, email string) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour)
	sessionData := map[string]interface{}{
		"uid":        uid,
		"email":      email,
		"active":     true,
		"expires_at": expiresAt,
	}
	sessionID, err := firestore.CreateDocument(ctx, "user_sessions", sessionData)
	return sessionID, expiresAt, err
}

func updateLastLogin(ctx context.Context, uid string) error {
	return firestore.UpdateDocument(ctx, "user_activity", uid, map[string]interface{}{
		"last_login": time.Now(),
	})
}