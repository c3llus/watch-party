package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"watch-party/pkg/config"
)

func newPgDB(
	cfg *config.Config,
) (*sql.DB, error) {
	dsn := getDSN(cfg.Database)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	// ping db to ensure the connection is alive and working
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getDSN(
	cfg config.DatabaseConfig,
) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)
}
