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

// prefix before all URLs:
const routePrefix = "/chatapp"

// default constants
const (
	defaultName     = "Jane Smith"
	defaultEmail    = "janes@yorku.ca"
	defaultTopic    = "chat"
	redisChannel    = "messages"
	redisIDKey      = "id"
	redisMessages   = "messages"
	redisAddress    = "127.0.0.1:6379"
	redisPassword   = ""
	postgresAddress = "postgresql://root:password@localhost:5433/root?sslmode=disable"
)

// Message struct for individaul messages
type Message struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Date    string `json:"date"`
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

// Helper function to convert string to JSON and return as a String
func (m *Message) toJSON() string {
	return fmt.Sprintf(`{"id":%d,"name":"%s","email":"%s","date":"%s","topic":"%s","content":"%s"}`,
		m.ID, m.Name, m.Email, m.Date, m.Topic, m.Content)
}

// Helper function to encode messages into JSON
func encodeMessages(messages []*Message) string {
	var encodedMessages []string
	for _, message := range messages {
		encodedMessages = append(encodedMessages, message.toJSON())
	}
	return "[" + strings.Join(encodedMessages, ",") + "]"
}

// Helper function to set CORS Header
func setCorsHeaders(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// Helper Function to connect to Redis
func connectRedis() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: redisPassword,
		DB:       0,
	})

	return redisClient
}

// Helper function to connect to Postgres
func connectPostgres() (*sql.DB, error) {
	db, err := sql.Open("postgres", postgresAddress)
	if err != nil {
		log.Println("Error connecting to PostgreSQL:", err)
	}
	return db, nil
}

// Function to send new messages
func sendMessage(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(&w) // to avoid CORS errors
	log.Printf("Received request to send message from %s\n", r.RemoteAddr)

	//extracting message from Client
	message := &Message{
		Name:    r.FormValue("name"),
		Email:   r.FormValue("email"),
		Topic:   r.FormValue("topic"),
		Content: r.FormValue("content"),
		Date:    time.Now().Format("01/02/2006 15:04:05"),
	}
	log.Printf("Message content: %s\n", message.Content)

	broadCastRedis(r, message) // Broadcast message to all open clients via Redis
	insertMessagePostgres(w, message)
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

// Function to Broadcast message to all open clients via Redis and insert into Redis as well
func broadCastRedis(r *http.Request, message *Message) {
	redisClient := connectRedis()

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

// Function to establish connection using websocket
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

	redisClient := connectRedis()
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

// Function to get chat history
func chatHistory(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(&w) // to avoid CORS errors

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Attempt to fetch chat history from Redis
	redisMessages, err := fetchChatHistoryFromRedis(r)
	if err == nil {
		// If successful, return the chat history retrieved from Redis
		w.Header().Set("Content-Type", "application/json")
		w.Write(redisMessages)
		return
	}

	// If fetching from Redis fails, fall back to fetching from PostgreSQL
	chatHistoryFromPostgres(w)
}

// Function to fetch chat history from Redis
func fetchChatHistoryFromRedis(r *http.Request) ([]byte, error) {
	log.Println("Fetching chat from Redis")
	redisClient := connectRedis()

	// Get chat history from Redis
	pickledMessages, err := redisClient.LRange(r.Context(), redisMessages, 0, -1).Result()
	if err != nil {
		log.Println("Error fetching chat history from Redis:", err)
		return nil, err
	}

	// Unmarshal pickled messages into Message structs
	var messages []*Message
	for _, pickledMessage := range pickledMessages {
		var message Message
		if err := json.Unmarshal([]byte(pickledMessage), &message); err != nil {
			log.Println("Error unmarshaling message:", err)
			continue
		}
		messages = append(messages, &message)
	}

	// Marshal messages to JSON
	jsonMessages, err := json.Marshal(messages)
	if err != nil {
		log.Println("Error marshaling chat history to JSON:", err)
		return nil, err
	}

	return jsonMessages, nil
}

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

func main() {
	http.HandleFunc(routePrefix+"/history", chatHistory)
	http.HandleFunc(routePrefix+"/send", sendMessage)
	http.HandleFunc(routePrefix+"/websocket", webSocketConnection)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
