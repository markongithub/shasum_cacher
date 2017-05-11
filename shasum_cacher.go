package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net/http"
	"strings"
)

type HashRequest struct {
	Message string `json:"message"`
}

type HashResponse struct {
	Digest string `json:"digest"`
}

var redisPool = poolRedisConnections()

func poolRedisConnections() *redis.Pool {
	return &redis.Pool{
		MaxActive: 50,
		MaxIdle:   10,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", ":6379")
		},
	}
}

//  Handles all interaction with the sha256 library.
func FormPostResponse(req HashRequest) HashResponse {
	shaBytes := sha256.Sum256([]byte(req.Message))
	shaSlice := shaBytes[:]
	return HashResponse{Digest: hex.EncodeToString(shaSlice)}
}

func StoreHash(message, digest string) error {
	redisConn := redisPool.Get()
	defer redisConn.Close()
	_, err := redisConn.Do("SET", digest, message)
	return err
}

func LookupHash(digest string) (string, error) {
	redisConn := redisPool.Get()
	defer redisConn.Close()
	message, err := redis.String(redisConn.Do("GET", digest))
	if err != nil {
		return "", fmt.Errorf("Redis lookup failed: %v", err)
	}
	return message, nil
}

func PostHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var parsedPost HashRequest
	if err := decoder.Decode(&parsedPost); err != nil {
		log.Printf("Failed decoding post data: %v", err)
		return
	}
	response := FormPostResponse(parsedPost)
	if err := StoreHash(parsedPost.Message, response.Digest); err != nil {
		http.Error(w, "Error writing to Redis.", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func ParseGetURL(req *http.Request) (string, error) {
	baseString := req.URL.String()
	substrings := strings.Split(baseString, "/")
	if len(substrings) != 3 {
		log.Printf("This URL is no good: %s", req.URL.String())
		return "", errors.New("URL should be of the form \"messages/<message>\"")
	}
	return substrings[2], nil
}

func GetHandler(w http.ResponseWriter, req *http.Request) {
	digest, err := ParseGetURL(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %v", err), 401)
		return
	}
	message, err := LookupHash(digest)
	if err != nil {
		if err.Error() == "Redis lookup failed: redigo: nil returned" {
			http.Error(w, "Message digest not found in cache.", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Lookup failed: %v", err), http.StatusInternalServerError)
		}
		return
	}
	response := HashRequest{Message: message}
	json.NewEncoder(w).Encode(response)
}

func MessageHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("Request method: %s", req.Method)
	if req.Method == "POST" {
		PostHandler(w, req)
	} else {
		GetHandler(w, req)
	}
}

func main() {

	http.HandleFunc("/messages", MessageHandler)
	http.HandleFunc("/messages/", MessageHandler)
	err := http.ListenAndServeTLS(":6443", "localhost.crt", "localhost.key", nil)
	if err != nil {
		log.Fatal("Failed to open HTTPS listener: ", err)
	}
}
