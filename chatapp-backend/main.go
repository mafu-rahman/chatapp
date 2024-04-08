package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

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
	redisAddress    = "redis:6379"
	redisPassword   = ""
	postgresAddress = "postgresql://root:password@postgresql:5432/root?sslmode=disable"
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

	err := broadCastRedis(r, message) // Broadcast message to all open clients via Redis
	if err != nil {
		log.Println(err)
	}
	insertMessagePostgres(w, message) //insert messages into postgres for persistence
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

func main() {
	initPostgresDB()
	http.HandleFunc(routePrefix+"/history", chatHistory)
	http.HandleFunc(routePrefix+"/send", sendMessage)
	http.HandleFunc(routePrefix+"/websocket", webSocketConnection)
	http.HandleFunc("/", chatHistory)

	address := ":30223"
	fmt.Printf("Starting server on address %s...\n", address)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
