package main

import (
	"context"
	"log"

	"github.com/andrescris/firestore/lib/firebase"
	"github.com/andrescris/firestore/lib/firebase/auth"
	"github.com/andrescris/firestore/lib/firebase/firestore"
)

func main() {
	// Inicializar Firebase
	if err := firebase.InitFirebaseFromEnv(); err != nil {
		log.Fatalf("Error initializing Firebase: %v", err)
	}
	defer firebase.Close()

	ctx := context.Background()

	log.Println("🚀 Iniciando ejemplo de Firebase con autenticación OTP...")

	// ===== GESTIÓN DE USUARIOS =====
	log.Println("\n🔐 === GESTIÓN DE USUARIOS ===")
	
	// Crear un usuario sin contraseña
	userRequest := firebase.CreateUserRequest{
		Email:       "daniela@example.com",
		DisplayName: "Daniela Mora",
	}

	user, err := auth.CreateUser(ctx, userRequest)
	if err != nil {
		log.Printf("⚠️  Usuario ya existe o error: %v. Intentando obtenerlo...", err)
		existingUser, getErr := auth.GetUserByEmail(ctx, userRequest.Email)
		if getErr != nil {
			log.Fatalf("Error getting existing user: %v", getErr)
		}
		user = existingUser
	}
	log.Printf("👤 Usuario creado/obtenido: %s (%s)", user.DisplayName, user.UID)
	
	// Establecer claims personalizados
	claims := map[string]interface{}{"role": "editor", "tier": "premium"}
	err = auth.SetCustomClaims(ctx, user.UID, claims)
	if err != nil {
		log.Printf("⚠️  Error setting custom claims: %v", err)
	} else {
		log.Printf("🏷️  Claims establecidos para el usuario: role=%s", claims["role"])
	}


	// ===== FLUJO DE LOGIN CON OTP =====
	log.Println("\n🔑 === FLUJO DE LOGIN CON OTP ===")

	// 1. Solicitar un OTP para el usuario
	otpRequest := firebase.RequestOTPRequest{Email: user.Email}
	otpResponse, err := auth.RequestOTP(ctx, otpRequest)
	if err != nil {
		log.Fatalf("Error solicitando OTP: %v", err)
	}
	log.Printf("📬 Respuesta de solicitud de OTP: %s", otpResponse.Message)


	// 2. Simulación para obtener el OTP para este ejemplo
	otpQuery := firebase.QueryOptions{
		Filters:  []firebase.QueryFilter{{Field: "email", Operator: "==", Value: user.Email}},
		OrderBy:  "created_at",
		OrderDir: "desc",
		Limit:    1,
	}
	otpDocs, err := firestore.QueryDocuments(ctx, "user_otps", otpQuery)
	if err != nil || len(otpDocs) == 0 {
		log.Fatalf("No se pudo obtener el OTP para el ejemplo.")
	}
	simulatedOTP := otpDocs[0].Data["otp"].(string)
	log.Printf("🤫 (Simulación) OTP obtenido para el ejemplo: %s", simulatedOTP)


	// 3. Realizar el login con el OTP
	loginRequest := firebase.LoginWithOTPRequest{
		Email: user.Email,
		OTP:   simulatedOTP,
	}
	loginResponse, err := auth.LoginWithOTP(ctx, loginRequest)
	if err != nil {
		log.Fatalf("Error en el login con OTP: %v", err)
	}

	if !loginResponse.Success {
		log.Fatalf("Login fallido: %s", loginResponse.Message)
	}
	
	log.Printf("🎉 ¡Login exitoso para %s!", loginResponse.User.DisplayName)
	log.Printf("   - Session ID: %s", loginResponse.SessionID)


	// ===== FUNCIONES ADICIONALES =====
	log.Println("\n🔍 === FUNCIONES ADICIONALES ===")
	
	// Verificar si usuario existe
	exists, _ := auth.UserExists(ctx, user.UID)
	log.Printf("✅ Usuario '%s' existe: %t", user.DisplayName, exists)

	// Contar usuarios totales
	totalUsers, _ := auth.GetUserCount(ctx)
	log.Printf("👥 Total de usuarios en el proyecto: %d", totalUsers)


	log.Println("\n✅ ¡Ejemplo OTP finalizado exitosamente!")
}