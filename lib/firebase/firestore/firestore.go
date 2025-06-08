package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	firebase "github.com/andrescris/firestore/lib/firebase"
)

// CreateDocument crea un nuevo documento en la colección especificada
func CreateDocument(ctx context.Context, collection string, data map[string]interface{}) (string, error) {
	client := firebase.GetFirestoreClient()

	// Agregar timestamps automáticamente
	now := time.Now()
	data["created_at"] = now
	data["updated_at"] = now

	docRef, _, err := client.Collection(collection).Add(ctx, data)
	if err != nil {
		return "", fmt.Errorf("failed to create document in collection '%s': %w", collection, err)
	}

	return docRef.ID, nil
}

// CreateDocumentWithID crea un documento con un ID específico
func CreateDocumentWithID(ctx context.Context, collection, docID string, data map[string]interface{}) error {
	client := firebase.GetFirestoreClient()

	// Agregar timestamps automáticamente
	now := time.Now()
	data["created_at"] = now
	data["updated_at"] = now

	_, err := client.Collection(collection).Doc(docID).Set(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to create document with ID '%s' in collection '%s': %w", docID, collection, err)
	}

	return nil
}

// GetDocument obtiene un documento por su ID
func GetDocument(ctx context.Context, collection, docID string) (*firebase.Document, error) {
	client := firebase.GetFirestoreClient()

	doc, err := client.Collection(collection).Doc(docID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get document '%s' from collection '%s': %w", docID, collection, err)
	}

	if !doc.Exists() {
		return nil, &firebase.DocumentNotFoundError{Collection: collection, DocumentID: docID}
	}

	return &firebase.Document{
		ID:   doc.Ref.ID,
		Data: doc.Data(),
	}, nil
}

// GetAllDocuments obtiene todos los documentos de una colección
func GetAllDocuments(ctx context.Context, collection string) ([]*firebase.Document, error) {
	client := firebase.GetFirestoreClient()

	iter := client.Collection(collection).Documents(ctx)
	defer iter.Stop()

	var documents []*firebase.Document

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate documents in collection '%s': %w", collection, err)
		}

		documents = append(documents, &firebase.Document{
			ID:   doc.Ref.ID,
			Data: doc.Data(),
		})
	}

	return documents, nil
}

// UpdateDocument actualiza un documento existente (merge completo)
func UpdateDocument(ctx context.Context, collection, docID string, data map[string]interface{}) error {
	client := firebase.GetFirestoreClient()

	// Agregar timestamp de actualización
	data["updated_at"] = time.Now()

	_, err := client.Collection(collection).Doc(docID).Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return fmt.Errorf("failed to update document '%s' in collection '%s': %w", docID, collection, err)
	}

	return nil
}

// UpdateDocumentFields actualiza campos específicos de un documento
func UpdateDocumentFields(ctx context.Context, collection, docID string, updates []firestore.Update) error {
	client := firebase.GetFirestoreClient()

	// Agregar timestamp de actualización
	updates = append(updates, firestore.Update{
		Path:  "updated_at",
		Value: time.Now(),
	})

	_, err := client.Collection(collection).Doc(docID).Update(ctx, updates)
	if err != nil {
		return fmt.Errorf("failed to update fields in document '%s' in collection '%s': %w", docID, collection, err)
	}

	return nil
}

// DeleteDocument elimina un documento
func DeleteDocument(ctx context.Context, collection, docID string) error {
	client := firebase.GetFirestoreClient()

	_, err := client.Collection(collection).Doc(docID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete document '%s' from collection '%s': %w", docID, collection, err)
	}

	return nil
}

// QueryDocuments realiza una consulta con filtros y opciones
func QueryDocuments(ctx context.Context, collection string, options firebase.QueryOptions) ([]*firebase.Document, error) {
	client := firebase.GetFirestoreClient()

	query := client.Collection(collection).Query

	// Aplicar filtros
	for _, filter := range options.Filters {
		query = query.Where(filter.Field, filter.Operator, filter.Value)
	}

	// Aplicar ordenamiento
	if options.OrderBy != "" {
		dir := firestore.Asc
		if options.OrderDir == "desc" {
			dir = firestore.Desc
		}
		query = query.OrderBy(options.OrderBy, dir)
	}

	// Aplicar offset
	if options.Offset > 0 {
		query = query.Offset(options.Offset)
	}

	// Aplicar límite
	if options.Limit > 0 {
		query = query.Limit(options.Limit)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	var documents []*firebase.Document

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query documents in collection '%s': %w", collection, err)
		}

		documents = append(documents, &firebase.Document{
			ID:   doc.Ref.ID,
			Data: doc.Data(),
		})
	}

	return documents, nil
}

// DocumentExists verifica si un documento existe
func DocumentExists(ctx context.Context, collection, docID string) (bool, error) {
	client := firebase.GetFirestoreClient()

	doc, err := client.Collection(collection).Doc(docID).Get(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check if document exists '%s' in collection '%s': %w", docID, collection, err)
	}

	return doc.Exists(), nil
}

// CountDocuments cuenta los documentos en una colección (con filtros opcionales)
func CountDocuments(ctx context.Context, collection string, filters []firebase.QueryFilter) (int, error) {
	client := firebase.GetFirestoreClient()

	query := client.Collection(collection).Query

	// Aplicar filtros
	for _, filter := range filters {
		query = query.Where(filter.Field, filter.Operator, filter.Value)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to count documents in collection '%s': %w", collection, err)
		}
		count++
	}

	return count, nil
}

// BatchWrite realiza operaciones en lote
func BatchWrite(ctx context.Context, operations []firebase.BatchOperation) error {
	client := firebase.GetFirestoreClient()
	batch := client.Batch()

	for _, op := range operations {
		switch op.Type {
		case "create":
			docRef := client.Collection(op.Collection).NewDoc()
			if op.DocumentID != "" {
				docRef = client.Collection(op.Collection).Doc(op.DocumentID)
			}

			// Agregar timestamps automáticamente
			now := time.Now()
			op.Data["created_at"] = now
			op.Data["updated_at"] = now

			batch.Set(docRef, op.Data)

		case "update":
			docRef := client.Collection(op.Collection).Doc(op.DocumentID)
			op.Data["updated_at"] = time.Now()
			batch.Set(docRef, op.Data, firestore.MergeAll)

		case "delete":
			docRef := client.Collection(op.Collection).Doc(op.DocumentID)
			batch.Delete(docRef)

		default:
			return fmt.Errorf("unsupported batch operation type: %s", op.Type)
		}
	}

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit batch operations: %w", err)
	}

	return nil
}