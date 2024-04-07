package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
)

// Helper Function to connect to Redis
func connectRedis() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: redisPassword,
		DB:       0,
	})

	return redisClient
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

// Function to fetch chat history from Redis
func fetchChatHistoryFromRedis(r *http.Request) ([]byte, error) {
	log.Println("Fetching chat from Redis")
	redisClient := connectRedis()

	// Get chat history from Redis
	fetchedMessages, err := redisClient.LRange(r.Context(), redisMessages, 0, -1).Result()
	if err != nil {
		log.Println("Error fetching chat history from Redis:", err)
		return nil, err
	}

	// Unmarshal fetched messages into Message structs
	var messages []*Message
	for _, messageStruct := range fetchedMessages {
		var message Message
		if err := json.Unmarshal([]byte(messageStruct), &message); err != nil {
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
