package database

import (
	"database/sql"
	"time"
)

// DB wraps the SQL database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database handler
func New(conn *sql.DB) *DB {
	return &DB{conn: conn}
}

// Message represents a message in the database
type Message struct {
	ID        int
	Content   string
	CreatedAt time.Time
}

// Provider represents a cloud provider
type Provider struct {
	ID        int
	Name      string
	CreatedAt time.Time
}

// Category represents a service category
type Category struct {
	ID          int
	Name        string
	Description *string
	CreatedAt   time.Time
}

// GetMessages retrieves all messages from the database
func (db *DB) GetMessages() ([]Message, error) {
	rows, err := db.conn.Query("SELECT id, content, created_at FROM messages ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// CreateMessage creates a new message in the database
func (db *DB) CreateMessage(content string) (*Message, error) {
	var msg Message
	err := db.conn.QueryRow(
		"INSERT INTO messages (content) VALUES ($1) RETURNING id, content, created_at",
		content,
	).Scan(&msg.ID, &msg.Content, &msg.CreatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &msg, nil
}

// GetProviders retrieves all providers from the database
func (db *DB) GetProviders() ([]Provider, error) {
	rows, err := db.conn.Query("SELECT id, name, created_at FROM providers ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}

	return providers, rows.Err()
}

// GetCategories retrieves all categories from the database
func (db *DB) GetCategories() ([]Category, error) {
	rows, err := db.conn.Query("SELECT id, name, description, created_at FROM service_categories ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, rows.Err()
}