package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

// If your application is not hosted on the root of your domain, apply this
// prefix before all URLs:
const routePrefix = "/chatapp"

const (
	defaultName   = "Jane Smith"
	defaultEmail  = "janes@yorku.ca"
	defaultTopic  = "chat"
	redisChannel  = "messages"
	redisIDKey    = "id"
	redisMessages = "messages"
)

// Message represents an individual sent message.
type Message struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Date    string `json:"date"`
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

func (m *Message) toJSON() string {
	return fmt.Sprintf(`{"id":%d,"name":"%s","email":"%s","date":"%s","topic":"%s","content":"%s"}`,
		m.ID, m.Name, m.Email, m.Date, m.Topic, m.Content)
}

func (m *Message) fromJSON(data string) error {
	return json.Unmarshal([]byte(data), m)
}

func encodeMessages(messages []*Message) string {
	var encodedMessages []string
	for _, message := range messages {
		encodedMessages = append(encodedMessages, message.toJSON())
	}
	return "[" + strings.Join(encodedMessages, ",") + "]"
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	log.Printf("Received request to send message from %s\n", r.RemoteAddr)

	message := &Message{
		Name:    r.FormValue("name"),
		Email:   r.FormValue("email"),
		Topic:   r.FormValue("topic"),
		Content: r.FormValue("content"),
		Date:    time.Now().Format("01/02/2006 15:04:05"),
	}

	log.Printf("Message content: %s\n", message.Content)

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", "postgresql://root:password@localhost:5433/root?sslmode=disable")
	if err != nil {
		log.Println("Error connecting to PostgreSQL:", err)
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

	// Now that we've recorded the message in PostgreSQL, broadcast it to all open clients via Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	id, err := redisClient.Incr(r.Context(), redisIDKey).Result()
	if err != nil {
		log.Println("Error incrementing ID:", err)
		return
	}
	message.ID = int(id)

	err = redisClient.RPush(r.Context(), redisMessages, message.toJSON()).Err()
	if err != nil {
		log.Println("Error pushing message to Redis:", err)
		return
	}

	err = redisClient.Publish(r.Context(), redisChannel, encodeMessages([]*Message{message})).Err()
	if err != nil {
		log.Println("Error publishing message to Redis channel:", err)
		return
	}
}

func webSocketConnection(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	pubsub := redisClient.Subscribe(r.Context(), redisChannel)
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(r.Context())
		if err != nil {
			log.Println("Error receiving message from Redis:", err)
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
			log.Println("Error writing message to WebSocket connection:", err)
			return
		}
	}
}

func chatHistory(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", "postgresql://root:password@localhost:5433/root?sslmode=disable")
	if err != nil {
		log.Println("Error connecting to PostgreSQL:", err)
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

func main() {
	http.HandleFunc(routePrefix+"/history", chatHistory)
	http.HandleFunc(routePrefix+"/send", sendMessage)
	http.HandleFunc(routePrefix+"/websocket", webSocketConnection)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
