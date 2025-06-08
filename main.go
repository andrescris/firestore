package main

import (
	"context"
	"fmt"
	"log"
	"time"

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

	log.Println("🚀 Iniciando ejemplo completo de Firebase (Firestore + Auth)...")

	// ===== AUTHENTICATION =====
	log.Println("\n🔐 === AUTHENTICATION ===")

	// Crear un usuario
	userRequest := firebase.CreateUserRequest{
		Email:       "ana@example.com",
		Password:    "password123",
		DisplayName: "Ana García",
		PhotoURL:    "https://example.com/avatar.jpg",
	}

	user, err := auth.CreateUser(ctx, userRequest)
	if err != nil {
		log.Printf("⚠️  Usuario ya existe o error: %v", err)
		// Intentar obtener el usuario existente
		existingUser, getErr := auth.GetUserByEmail(ctx, userRequest.Email)
		if getErr != nil {
			log.Fatalf("Error getting existing user: %v", getErr)
		}
		user = existingUser
	}

	log.Printf("👤 Usuario: %s (%s)", user.DisplayName, user.UID)

	// Establecer claims personalizados (roles)
	claims := map[string]interface{}{
		"role":        "admin",
		"permissions": []string{"read", "write", "delete"},
		"department":  "IT",
	}

	err = auth.SetCustomClaims(ctx, user.UID, claims)
	if err != nil {
		log.Printf("⚠️  Error setting custom claims: %v", err)
	} else {
		log.Printf("🏷️  Claims establecidos para el usuario")
	}

	// Crear token personalizado
	customToken, err := auth.CreateCustomToken(ctx, user.UID, claims)
	if err != nil {
		log.Printf("⚠️  Error creating custom token: %v", err)
	} else {
		log.Printf("🎫 Token personalizado creado (longitud: %d)", len(customToken))
	}

	// Listar algunos usuarios (máximo 5)
	users, nextToken, err := auth.ListUsers(ctx, 5, "")
	if err != nil {
		log.Printf("⚠️  Error listing users: %v", err)
	} else {
		log.Printf("👥 Usuarios encontrados: %d", len(users))
		if nextToken != "" {
			log.Printf("📄 Hay más páginas disponibles")
		}
		for i, u := range users {
			log.Printf("   %d. %s (%s)", i+1, u.DisplayName, u.Email)
		}
	}

	// ===== FIRESTORE =====
	log.Println("\n📄 === FIRESTORE ===")

	// Crear perfil del usuario en Firestore
	profileData := map[string]interface{}{
		"user_id":     user.UID,
		"email":       user.Email,
		"plan":        "premium",
		"preferences": map[string]interface{}{
			"notifications": true,
			"theme":         "dark",
			"language":      "es",
		},
		"last_login": time.Now(),
	}

	profileID, err := firestore.CreateDocument(ctx, "profiles", profileData)
	if err != nil {
		log.Fatalf("Error creating profile: %v", err)
	}

	log.Printf("📋 Perfil creado con ID: %s", profileID)

	// Crear algunos documentos de prueba
	for i := 1; i <= 3; i++ {
		postData := map[string]interface{}{
			"title":   fmt.Sprintf("Post %d", i),
			"content": fmt.Sprintf("Contenido del post número %d", i),
			"author":  user.UID,
			"views":   i * 10,
			"status":  "published",
		}

		postID, err := firestore.CreateDocument(ctx, "posts", postData)
		if err != nil {
			log.Printf("⚠️  Error creating post %d: %v", i, err)
		} else {
			log.Printf("📝 Post %d creado con ID: %s", i, postID)
		}
	}

	// Consultar posts del usuario
	queryOptions := firebase.QueryOptions{
		Filters: []firebase.QueryFilter{
			{Field: "author", Operator: "==", Value: user.UID},
			{Field: "status", Operator: "==", Value: "published"},
		},
		OrderBy:  "views",
		OrderDir: "desc",
		Limit:    10,
	}

	userPosts, err := firestore.QueryDocuments(ctx, "posts", queryOptions)
	if err != nil {
		log.Printf("⚠️  Error querying posts: %v", err)
	} else {
		log.Printf("📚 Posts del usuario encontrados: %d", len(userPosts))
		for _, post := range userPosts {
			title := post.Data["title"]
			views := post.Data["views"]
			log.Printf("   - %s (%v views)", title, views)
		}
	}

	// ===== OPERACIONES EN LOTE =====
	log.Println("\n⚡ === BATCH OPERATIONS ===")

	// Crear múltiples documentos en una operación en lote
	batchOps := []firebase.BatchOperation{
		{
			Type:       "create",
			Collection: "analytics",
			Data: map[string]interface{}{
				"user_id": user.UID,
				"event":   "user_created",
				"data":    map[string]interface{}{"plan": "premium"},
			},
		},
		{
			Type:       "create",
			Collection: "analytics",
			Data: map[string]interface{}{
				"user_id": user.UID,
				"event":   "profile_completed",
				"data":    map[string]interface{}{"fields": 5},
			},
		},
		{
			Type:       "update",
			Collection: "profiles",
			DocumentID: profileID,
			Data: map[string]interface{}{
				"setup_completed": true,
				"analytics_count": 2,
			},
		},
	}

	err = firestore.BatchWrite(ctx, batchOps)
	if err != nil {
		log.Printf("⚠️  Error in batch operations: %v", err)
	} else {
		log.Printf("📦 Operaciones en lote completadas (%d ops)", len(batchOps))
	}

	// ===== FUNCIONES ADICIONALES =====
	log.Println("\n🔍 === FUNCIONES ADICIONALES ===")

	// Verificar si usuario existe
	exists, err := auth.UserExists(ctx, user.UID)
	if err != nil {
		log.Printf("⚠️  Error checking user existence: %v", err)
	} else {
		log.Printf("✅ Usuario existe: %t", exists)
	}

	// Verificar si documento existe
	docExists, err := firestore.DocumentExists(ctx, "profiles", profileID)
	if err != nil {
		log.Printf("⚠️  Error checking document existence: %v", err)
	} else {
		log.Printf("✅ Perfil existe: %t", docExists)
	}

	// Contar documentos
	postCount, err := firestore.CountDocuments(ctx, "posts", []firebase.QueryFilter{
		{Field: "author", Operator: "==", Value: user.UID},
	})
	if err != nil {
		log.Printf("⚠️  Error counting posts: %v", err)
	} else {
		log.Printf("📊 Total posts del usuario: %d", postCount)
	}

	// Contar usuarios totales
	totalUsers, err := auth.GetUserCount(ctx)
	if err != nil {
		log.Printf("⚠️  Error counting users: %v", err)
	} else {
		log.Printf("👥 Total de usuarios: %d", totalUsers)
	}

	// ===== RESUMEN FINAL =====
	log.Println("\n🎉 === RESUMEN FINAL ===")
	log.Printf("👤 Usuario: %s (%s)", user.DisplayName, user.UID)
	log.Printf("📧 Email: %s", user.Email)
	log.Printf("📋 Perfil ID: %s", profileID)
	log.Printf("📝 Posts creados: %d", len(userPosts))
	log.Printf("🏷️  Claims: role=%s", claims["role"])
	log.Printf("👥 Total usuarios: %d", totalUsers)
	log.Printf("📊 Posts del usuario: %d", postCount)

	log.Println("\n✅ ¡Ejemplo completo finalizado exitosamente!")
	log.Println("🔥 Firebase package funcionando con Firestore + Auth")
}