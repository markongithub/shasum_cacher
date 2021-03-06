package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"net/http"
	"strings"
)

var (
	httpsServerAddress = flag.String("https_server_address", ":5000", "Address on which to serve HTTPS requests")
	redisServerAddress = flag.String("redis_server_address", "redis:6379", "Redis server where message digests are cached")
	serverSSLKey       = flag.String("server_ssl_key", "localhost.key", "private key for HTTPS server")
	serverSSLCert      = flag.String("server_ssl_cert", "localhost.crt", "signed certificate for HTTPS server")
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
			return redis.Dial("tcp", *redisServerAddress)
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

func LoggingHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("%s %s %s", req.RemoteAddr, req.Method, req.URL)
	if req.Method == "POST" {
		PostHandler(w, req)
	} else {
		GetHandler(w, req)
	}
}

func main() {
	flag.Parse()

	http.HandleFunc("/messages", LoggingHandler)
	http.HandleFunc("/messages/", LoggingHandler)
        log.Printf("About to listen on port %v", *httpsServerAddress)
	err := http.ListenAndServeTLS(*httpsServerAddress, *serverSSLCert, *serverSSLKey, nil)
	if err != nil {
		log.Fatal("Failed to open HTTPS listener: ", err)
	}
}
