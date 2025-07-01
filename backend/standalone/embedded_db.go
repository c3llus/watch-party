package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"watch-party/pkg/config"
	"watch-party/pkg/database"
	"watch-party/pkg/logger"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

//go:embed schema.sql
var schemaSQL string

var (
	embeddedDB   *embeddedpostgres.EmbeddedPostgres
	dbConnection *sql.DB
	dbPort       uint32
)

// findAvailablePort finds an available port starting from the given port
func findAvailablePort(startPort uint32) uint32 {
	for port := startPort; port < startPort+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	log.Fatalf("Could not find an available port starting from %d", startPort)
	return 0
}

func startEmbeddedDB(ctx context.Context) {
	logger.Info("Starting embedded PostgreSQL 17...")

	// Find an available port starting from 15432
	dbPort = findAvailablePort(15432)
	logger.Info(fmt.Sprintf("Using port %d for PostgreSQL", dbPort))

	// create data directory in user home
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}

	dataDir := filepath.Join(homeDir, ".watch-party", "data")
	runtimeDir := filepath.Join(homeDir, ".watch-party", "runtime")
	binariesDir := filepath.Join(homeDir, ".watch-party", "binaries")

	// create directories
	for _, dir := range []string{dataDir, runtimeDir, binariesDir} {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// clean up any existing data to avoid conflicts
	err = os.RemoveAll(dataDir)
	if err != nil {
		logger.Info(fmt.Sprintf("Warning: Failed to clean up existing data directory: %v", err))
	}
	err = os.MkdirAll(dataDir, 0755)
	if err != nil {
		log.Fatalf("Failed to recreate data directory: %v", err)
	}

	// create embedded PostgreSQL instance with dynamic port
	embeddedDB = embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username("postgres").
		Password("postgres").
		Database("watchparty").
		Port(dbPort).
		RuntimePath(runtimeDir).
		DataPath(dataDir).
		BinariesPath(binariesDir))

	// start the database
	err = embeddedDB.Start()
	if err != nil {
		log.Fatalf("Failed to start embedded PostgreSQL: %v", err)
	}

	logger.Info("Waiting for embedded PostgreSQL to be ready...")

	// wait for database to be ready with retries
	for i := 0; i < 30; i++ { // try for 30 seconds
		time.Sleep(1 * time.Second)

		// try to connect
		connectionString := fmt.Sprintf("host=localhost port=%d user=postgres password=postgres dbname=watchparty sslmode=disable", dbPort)
		testDB, err := sql.Open("postgres", connectionString)
		if err == nil {
			err := testDB.Ping()
			if err == nil {
				testDB.Close()
				logger.Info("✅ Embedded PostgreSQL is ready!")
				break
			}
			testDB.Close()
		}

		if i == 29 {
			log.Fatalf("Embedded PostgreSQL failed to start after 30 seconds")
		}

		logger.Info(fmt.Sprintf("Waiting for PostgreSQL... (%d/30)", i+1))
	}

	// create config for database connection
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:            "localhost",
			Port:            fmt.Sprintf("%d", dbPort),
			Username:        "postgres",
			Password:        "postgres",
			Name:            "watchparty",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    25,
			ConnMaxLifetime: config.Duration(5 * time.Minute),
		},
	}

	// connect using your existing database package
	dbConnection, err = database.NewPgDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to embedded PostgreSQL: %v", err)
	}

	// test connection
	err = dbConnection.Ping()
	if err != nil {
		log.Fatalf("Failed to ping embedded PostgreSQL: %v", err)
	}

	// initialize schema using embedded schema file or existing schema
	err = initializeSchema()
	if err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	logger.Info(fmt.Sprintf("✅ Embedded PostgreSQL 17 started successfully on port %d", dbPort))

	<-ctx.Done()

	logger.Info("Shutting down embedded PostgreSQL...")
	if dbConnection != nil {
		dbConnection.Close()
	}
	if embeddedDB != nil {
		embeddedDB.Stop()
	}
}

func initializeSchema() error {
	logger.Info("Initializing database schema...")

	if len(schemaSQL) == 0 {
		return fmt.Errorf("schema.sql is empty or not embedded properly")
	}

	_, err := dbConnection.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	logger.Info("✅ Database schema initialized successfully")
	return nil
}

// GetDBConnection returns the database connection for use by services
func GetDBConnection() *sql.DB {
	return dbConnection
}

// GetDBAddr returns the address of the embedded PostgreSQL instance
func GetDBAddr() string {
	return fmt.Sprintf("localhost:%d", dbPort)
}

// GetDBPort returns the port of the embedded PostgreSQL instance
func GetDBPort() uint32 {
	return dbPort
}
