package firebase

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

var (
	app             *firebase.App
	firestoreClient *firestore.Client
	authClient      *auth.Client
	once            sync.Once
	initErr         error
	projectID       string
)

// InitFirebaseFromEnv inicializa Firestore y Auth desde variables de entorno
func InitFirebaseFromEnv() error {
	once.Do(func() {
		// Cargar archivo .env si existe
		if err := godotenv.Load(); err != nil {
			// No es un error crítico si no existe .env
			fmt.Println("No .env file found, using system environment variables")
		}

		var credJSON string
		var opt option.ClientOption

		// Intentar obtener credenciales de diferentes fuentes
		credJSON = os.Getenv("FIREBASE_SERVICE_ACCOUNT")
		
		if credJSON != "" {
			// Opción 1: JSON directo desde variable de entorno
			opt = option.WithCredentialsJSON([]byte(credJSON))
		} else if credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credFile != "" {
			// Opción 2: Archivo de credenciales
			opt = option.WithCredentialsFile(credFile)
			
			// Leer el archivo para obtener el project_id
			fileData, err := ioutil.ReadFile(credFile)
			if err != nil {
				initErr = fmt.Errorf("failed to read credentials file: %w", err)
				return
			}
			credJSON = string(fileData)
		} else {
			// Opción 3: Buscar archivo en ubicaciones comunes
			commonPaths := []string{
				"firebase-credentials.json",
				"service-account-key.json",
				"firebase-service-account.json",
				"credentials.json",
			}
			
			for _, path := range commonPaths {
				if _, err := os.Stat(path); err == nil {
					fileData, err := ioutil.ReadFile(path)
					if err != nil {
						continue
					}
					credJSON = string(fileData)
					opt = option.WithCredentialsFile(path)
					break
				}
			}
			
			if credJSON == "" {
				initErr = ErrMissingCredentials
				return
			}
		}

		// Extraer project_id de las credenciales
		var credMap map[string]interface{}
		if err := json.Unmarshal([]byte(credJSON), &credMap); err != nil {
			initErr = fmt.Errorf("invalid JSON in credentials: %w", err)
			return
		}

		if pid, ok := credMap["project_id"].(string); ok {
			projectID = pid
		} else {
			initErr = fmt.Errorf("project_id not found in credentials")
			return
		}

		ctx := context.Background()

		// Inicializar Firebase App
		firebaseApp, err := firebase.NewApp(ctx, nil, opt)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize Firebase app: %w", err)
			return
		}
		app = firebaseApp

		// Inicializar clientes
		initErr = initializeClients(ctx)
	})

	return initErr
}

func initializeClients(ctx context.Context) error {
	var err error

	// Firestore
	firestoreClient, err = app.Firestore(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %w", err)
	}

	// Auth
	authClient, err = app.Auth(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Auth client: %w", err)
	}

	return nil
}

// GetFirestoreClient retorna el cliente de Firestore
func GetFirestoreClient() *firestore.Client {
	if firestoreClient == nil {
		panic("Firestore client not initialized. Call InitFirebaseFromEnv first.")
	}
	return firestoreClient
}

// GetAuthClient retorna el cliente de Auth
func GetAuthClient() *auth.Client {
	if authClient == nil {
		panic("Auth client not initialized. Call InitFirebaseFromEnv first.")
	}
	return authClient
}

// GetProjectID retorna el ID del proyecto
func GetProjectID() string {
	return projectID
}

// GetClient alias para mantener compatibilidad con tu implementación original
func GetClient() *firestore.Client {
	return GetFirestoreClient()
}

// Close cierra todas las conexiones
func Close() error {
	if firestoreClient != nil {
		if err := firestoreClient.Close(); err != nil {
			return fmt.Errorf("firestore close error: %w", err)
		}
	}
	return nil
}