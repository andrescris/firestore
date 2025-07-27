package firebase

import "time"

// Document representa un documento de Firestore con su ID
type Document struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

// QueryFilter representa un filtro para consultas de Firestore
type QueryFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // "==", "!=", ">", ">=", "<", "<=", "in", "array-contains"
	Value    interface{} `json:"value"`
}

// QueryOptions representa opciones para consultas de Firestore
type QueryOptions struct {
	Filters  []QueryFilter `json:"filters,omitempty"`
	OrderBy  string        `json:"order_by,omitempty"`
	OrderDir string        `json:"order_dir,omitempty"` // "asc" or "desc"
	Limit    int           `json:"limit,omitempty"`
	Offset   int           `json:"offset,omitempty"`
}

// BatchOperation representa una operación en lote para Firestore
type BatchOperation struct {
	Type       string                 `json:"type"`        // "create", "update", "delete"
	Collection string                 `json:"collection"`
	DocumentID string                 `json:"document_id,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// UserRecord información de usuario de Auth
type UserRecord struct {
	UID           string                 `json:"uid"`
	Email         string                 `json:"email"`
	DisplayName   string                 `json:"display_name,omitempty"`
	PhotoURL      string                 `json:"photo_url,omitempty"`
	Disabled      bool                   `json:"disabled"`
	EmailVerified bool                   `json:"email_verified"`
	CustomClaims  map[string]interface{} `json:"custom_claims,omitempty"`
	CreationTime  time.Time              `json:"creation_time"`
	LastLogInTime time.Time              `json:"last_login_time"`
}

// CreateUserRequest solicitud para crear usuario (solo con email)
type CreateUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	PhotoURL    string `json:"photo_url,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

// UpdateUserRequest solicitud para actualizar usuario
type UpdateUserRequest struct {
	Email         *string                `json:"email,omitempty"`
	DisplayName   *string                `json:"display_name,omitempty"`
	PhotoURL      *string                `json:"photo_url,omitempty"`
	Disabled      *bool                  `json:"disabled,omitempty"`
	EmailVerified *bool                  `json:"email_verified,omitempty"`
	CustomClaims  map[string]interface{} `json:"custom_claims,omitempty"`
}

// RequestOTPRequest solicitud para pedir un OTP
type RequestOTPRequest struct {
	Email string `json:"email"`
}

// LoginWithOTPRequest solicitud de login con OTP
type LoginWithOTPRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

// IncrementValue representa un valor de incremento para Firestore
type IncrementValue int

// Helper para crear un valor de incremento
func Increment(value int) IncrementValue {
	return IncrementValue(value)
}