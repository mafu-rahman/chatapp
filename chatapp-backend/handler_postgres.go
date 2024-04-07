package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Helper function to connect to Postgres
func connectPostgres() (*sql.DB, error) {
	db, err := sql.Open("postgres", postgresAddress)
	if err != nil {
		log.Println("Error connecting to PostgreSQL:", err)
	}
	return db, nil
}

// Function for Postgres
func insertMessagePostgres(w http.ResponseWriter, message *Message) {
	// Connect to PostgreSQL
	db, err := connectPostgres()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Insert message into PostgreSQL
	_, err = db.Exec("INSERT INTO messages (name, email, topic, content, date) VALUES ($1, $2, $3, $4, $5)",
		message.Name, message.Email, message.Topic, message.Content, message.Date)
	if err != nil {
		log.Println("Error inserting message into PostgreSQL:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// Function to fectch chat hostory from Postgres
func chatHistoryFromPostgres(w http.ResponseWriter) {
	log.Println("Fetching chat from Postgres")
	// Connect to PostgreSQL
	db, err := connectPostgres()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Retrieve chat history from PostgreSQL
	rows, err := db.Query("SELECT id, name, email, topic, content, date FROM messages")
	if err != nil {
		log.Println("Error retrieving chat history from PostgreSQL:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Iterate over rows and construct messages
	var messages []*Message
	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.ID, &message.Name, &message.Email, &message.Topic, &message.Content, &message.Date); err != nil {
			log.Println("Error scanning row from PostgreSQL:", err)
			continue
		}
		messages = append(messages, &message)
	}

	// Marshal messages to JSON
	jsonMessages, err := json.Marshal(messages)
	if err != nil {
		log.Println("Error marshaling chat history to JSON:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set Content-Type header and write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonMessages)
}

// Initializing the postgres database
func initPostgresDB() error {
	// Connect to PostgreSQL
	db, err := connectPostgres()
	if err != nil {
		log.Fatal("Error connecting to PostgreSQL:", err)
	}
	defer db.Close()

	// Create messages table if not exists
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255),
			email VARCHAR(255),
			date TIMESTAMP,
			topic VARCHAR(255),
			content TEXT
	)`)
	if err != nil {
		log.Fatal("Error creating messages table:", err)
	}

	fmt.Println("PostgreSQL database initialized successfully")

	return nil
}
