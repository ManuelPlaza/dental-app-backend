package arangodb

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
)

// Client envuelve la conexión a ArangoDB con el manejo de la base de datos por defecto.
type Client struct {
	DB arangodb.Database
}

// NewClient se conecta a ArangoDB y garantiza que existan la base de datos
// y las colecciones de vértices y aristas que usa el dominio.
func NewClient() (*Client, error) {
	url := os.Getenv("ARANGO_URL")
	user := os.Getenv("ARANGO_USER")
	pass := os.Getenv("ARANGO_PASSWORD")
	dbName := os.Getenv("ARANGO_DATABASE")

	if url == "" || dbName == "" {
		log.Println("[ArangoDB] modo simulación — variables vacías, no se conecta")
		return &Client{}, nil
	}

	endpoint := connection.NewRoundRobinEndpoints([]string{url})
	conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, false))
	conn.SetAuthentication(connection.NewBasicAuth(user, pass))

	client := arangodb.NewClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Crear base de datos si no existe
	exists, err := client.DatabaseExists(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("arangodb: check db: %w", err)
	}
	if !exists {
		if _, err := client.CreateDatabase(ctx, dbName, nil); err != nil {
			return nil, fmt.Errorf("arangodb: create db: %w", err)
		}
		log.Printf("[ArangoDB] base de datos %q creada", dbName)
	}

	db, err := client.GetDatabase(ctx, dbName, nil)
	if err != nil {
		return nil, fmt.Errorf("arangodb: get db: %w", err)
	}

	if err := ensureCollections(ctx, db); err != nil {
		return nil, err
	}

	log.Printf("[ArangoDB] conectado a %s/%s", url, dbName)
	return &Client{DB: db}, nil
}

// ensureCollections crea las colecciones del grafo de referidos si no existen.
// Vértices: patients (un nodo por paciente).
// Aristas:  referred (de referidor → referido).
func ensureCollections(ctx context.Context, db arangodb.Database) error {
	if err := createCollection(ctx, db, "patients", arangodb.CollectionTypeDocument); err != nil {
		return err
	}
	if err := createCollection(ctx, db, "referred", arangodb.CollectionTypeEdge); err != nil {
		return err
	}
	return nil
}

func createCollection(ctx context.Context, db arangodb.Database, name string, collType arangodb.CollectionType) error {
	exists, err := db.CollectionExists(ctx, name)
	if err != nil {
		return fmt.Errorf("arangodb: check %s: %w", name, err)
	}
	if exists {
		return nil
	}
	opts := &arangodb.CreateCollectionPropertiesV2{Type: &collType}
	if _, err := db.CreateCollectionV2(ctx, name, opts); err != nil {
		return fmt.Errorf("arangodb: create %s: %w", name, err)
	}
	return nil
}
