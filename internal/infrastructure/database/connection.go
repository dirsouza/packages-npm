package database

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// Connection wraps the database handle.
type Connection struct {
	DB *sql.DB
}

// Open creates a new database connection for the given DSN.
func Open(dsn string) (*Connection, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite does not support concurrent writers
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Connection{DB: db}, nil
}

func (c *Connection) Close() error { return Close(c) }

// Close releases the provided database connection.
func Close(conn *Connection) error {
	if conn == nil || conn.DB == nil {
		return nil
	}
	return conn.DB.Close()
}
