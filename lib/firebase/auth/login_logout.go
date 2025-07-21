package auth

import (
	"context"
	"fmt"
	"time"

	firebase "github.com/andrescris/firestore/lib/firebase"
	"github.com/andrescris/firestore/lib/firebase/firestore"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest estructura para solicitud de login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

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

// LogoutRequest estructura para solicitud de logout
type LogoutRequest struct {
	UID       string `json:"uid"`
	SessionID string `json:"session_id,omitempty"`
}

// LogoutResponse respuesta del logout
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SessionInfo información de sesión
type SessionInfo struct {
	SessionID string                 `json:"session_id"`
	UID       string                 `json:"uid"`
	Email     string                 `json:"email"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	Active    bool                   `json:"active"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Login autentica un usuario con email/password
func Login(ctx context.Context, request LoginRequest) (*LoginResponse, error) {
	// 1. Validar que el usuario existe
	user, err := GetUserByEmail(ctx, request.Email)
	if err != nil {
		return &LoginResponse{
			Success: false,
			Message: "Usuario no encontrado o credenciales inválidas",
		}, nil
	}

	// 2. Verificar si el usuario está activo
	if user.Disabled {
		return &LoginResponse{
			Success: false,
			Message: "Cuenta desactivada. Contacta al administrador",
		}, nil
	}

	// 3. Verificar password (necesitas almacenar hash en Firestore)
	valid, err := verifyPassword(ctx, user.UID, request.Password)
	if err != nil {
		return nil, fmt.Errorf("error verifying password: %w", err)
	}
	
	if !valid {
		return &LoginResponse{
			Success: false,
			Message: "Credenciales inválidas",
		}, nil
	}

	// 4. Obtener claims del usuario (roles, permisos)
	claims, err := getUserClaims(ctx, user.UID)
	if err != nil {
		// Si no hay claims, usar básicos
		claims = map[string]interface{}{
			"role": "user",
		}
	}

	// 5. Crear token personalizado
	customToken, err := CreateCustomToken(ctx, user.UID, claims)
	if err != nil {
		return nil, fmt.Errorf("error creating custom token: %w", err)
	}

	// 6. Crear sesión
	sessionID, expiresAt, err := createSession(ctx, user.UID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}

	// 7. Actualizar último login
	err = updateLastLogin(ctx, user.UID)
	if err != nil {
		// No es crítico si falla
		fmt.Printf("Warning: failed to update last login: %v", err)
	}

	return &LoginResponse{
		Success:     true,
		Message:     "Login exitoso",
		User:        user,
		CustomToken: customToken,
		SessionID:   sessionID,
		ExpiresAt:   expiresAt,
		Claims:      claims,
	}, nil
}

// Logout cierra la sesión del usuario
func Logout(ctx context.Context, request LogoutRequest) (*LogoutResponse, error) {
	// 1. Invalidar sesión si se proporciona sessionID
	if request.SessionID != "" {
		err := invalidateSession(ctx, request.SessionID)
		if err != nil {
			return nil, fmt.Errorf("error invalidating session: %w", err)
		}
	}

	// 2. Invalidar todas las sesiones del usuario (opcional)
	err := invalidateAllUserSessions(ctx, request.UID)
	if err != nil {
		return nil, fmt.Errorf("error invalidating user sessions: %w", err)
	}

	// 3. Actualizar último logout
	err = updateLastLogout(ctx, request.UID)
	if err != nil {
		// No es crítico si falla
		fmt.Printf("Warning: failed to update last logout: %v", err)
	}

	return &LogoutResponse{
		Success: true,
		Message: "Logout exitoso",
	}, nil
}

// ValidateSession verifica si una sesión es válida
func ValidateSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	session, err := getSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Verificar si la sesión ha expirado
	if time.Now().After(session.ExpiresAt) {
		// Invalidar sesión expirada
		invalidateSession(ctx, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	// Verificar si está activa
	if !session.Active {
		return nil, fmt.Errorf("session inactive")
	}

	return session, nil
}

// RefreshSession renueva una sesión existente
func RefreshSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	session, err := ValidateSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Extender expiración
	newExpiresAt := time.Now().Add(24 * time.Hour) // 24 horas

	err = updateSessionExpiration(ctx, sessionID, newExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("error refreshing session: %w", err)
	}

	session.ExpiresAt = newExpiresAt
	return session, nil
}

// === FUNCIONES AUXILIARES ===

// verifyPassword verifica el password del usuario
func verifyPassword(ctx context.Context, uid, password string) (bool, error) {
	// Obtener hash del password desde Firestore
	doc, err := firestore.GetDocument(ctx, "user_credentials", uid)
	if err != nil {
		return false, err
	}

	hashedPassword, ok := doc.Data["password_hash"].(string)
	if !ok {
		return false, fmt.Errorf("password hash not found")
	}

	// Comparar password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil, nil
}

// getUserClaims obtiene los claims del usuario desde Firestore
func getUserClaims(ctx context.Context, uid string) (map[string]interface{}, error) {
	doc, err := firestore.GetDocument(ctx, "user_claims", uid)
	if err != nil {
		return nil, err
	}

	claims, ok := doc.Data["claims"].(map[string]interface{})
	if !ok {
		return map[string]interface{}{"role": "user"}, nil
	}

	return claims, nil
}

// createSession crea una nueva sesión
func createSession(ctx context.Context, uid, email string) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour) // 24 horas
	
	sessionData := map[string]interface{}{
		"uid":       uid,
		"email":     email,
		"active":    true,
		"expires_at": expiresAt,
		"metadata": map[string]interface{}{
			"ip":         "unknown", // Puedes agregar IP real
			"user_agent": "unknown", // Puedes agregar user agent real
		},
	}

	sessionID, err := firestore.CreateDocument(ctx, "user_sessions", sessionData)
	if err != nil {
		return "", time.Time{}, err
	}

	return sessionID, expiresAt, nil
}

// getSession obtiene información de una sesión
func getSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	doc, err := firestore.GetDocument(ctx, "user_sessions", sessionID)
	if err != nil {
		return nil, err
	}

	session := &SessionInfo{
		SessionID: sessionID,
		UID:       doc.Data["uid"].(string),
		Email:     doc.Data["email"].(string),
		Active:    doc.Data["active"].(bool),
		CreatedAt: doc.Data["created_at"].(time.Time),
		ExpiresAt: doc.Data["expires_at"].(time.Time),
	}

	if metadata, ok := doc.Data["metadata"].(map[string]interface{}); ok {
		session.Metadata = metadata
	}

	return session, nil
}

// invalidateSession invalida una sesión específica
func invalidateSession(ctx context.Context, sessionID string) error {
	return firestore.UpdateDocument(ctx, "user_sessions", sessionID, map[string]interface{}{
		"active":    false,
		"ended_at":  time.Now(),
	})
}

// invalidateAllUserSessions invalida todas las sesiones de un usuario
func invalidateAllUserSessions(ctx context.Context, uid string) error {
	// Buscar todas las sesiones activas del usuario
	sessions, err := firestore.QueryDocuments(ctx, "user_sessions", firebase.QueryOptions{
		Filters: []firebase.QueryFilter{
			{Field: "uid", Operator: "==", Value: uid},
			{Field: "active", Operator: "==", Value: true},
		},
	})
	if err != nil {
		return err
	}

	// Invalidar todas las sesiones
	var batchOps []firebase.BatchOperation
	for _, session := range sessions {
		batchOps = append(batchOps, firebase.BatchOperation{
			Type:       "update",
			Collection: "user_sessions",
			DocumentID: session.ID,
			Data: map[string]interface{}{
				"active":   false,
				"ended_at": time.Now(),
			},
		})
	}

	if len(batchOps) > 0 {
		return firestore.BatchWrite(ctx, batchOps)
	}

	return nil
}

// updateLastLogin actualiza el timestamp del último login
func updateLastLogin(ctx context.Context, uid string) error {
	return firestore.UpdateDocument(ctx, "user_activity", uid, map[string]interface{}{
		"last_login": time.Now(),
		"login_count": firebase.IncrementValue(1), // Si usas increment
	})
}

// updateLastLogout actualiza el timestamp del último logout
func updateLastLogout(ctx context.Context, uid string) error {
	return firestore.UpdateDocument(ctx, "user_activity", uid, map[string]interface{}{
		"last_logout": time.Now(),
	})
}

// updateSessionExpiration actualiza la expiración de una sesión
func updateSessionExpiration(ctx context.Context, sessionID string, expiresAt time.Time) error {
	return firestore.UpdateDocument(ctx, "user_sessions", sessionID, map[string]interface{}{
		"expires_at": expiresAt,
	})
}

// HashPassword genera un hash bcrypt del password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// StoreUserCredentials almacena las credenciales del usuario
func StoreUserCredentials(ctx context.Context, uid, password string) error {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	return firestore.CreateDocumentWithID(ctx, "user_credentials", uid, map[string]interface{}{
		"password_hash": hashedPassword,
	})
}