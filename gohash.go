package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
        "github.com/garyburd/redigo/redis"
)

type HashRequest struct {
	Message string `json:"message"`
}

type HashResponse struct {
	Digest string `json:"digest"`
}

//  Handles all interaction with the sha256 library.
func FormPostResponse(req HashRequest) HashResponse {
	shaBytes := sha256.Sum256([]byte(req.Message))
	shaSlice := shaBytes[:]
	return HashResponse{Digest: hex.EncodeToString(shaSlice)}
}

func PostHandler(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var parsedPost HashRequest
	if err := decoder.Decode(&parsedPost); err != nil {
		log.Printf("Failed decoding post data: %v", err)
		return
	}
	json.NewEncoder(w).Encode(FormPostResponse(parsedPost))
}

func MessageHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("Request method: %s", req.Method)
	if req.Method == "POST" {
		PostHandler(w, req)
	} else {
		http.Error(w, "fail", 500)
	}
}

func main() {
	http.HandleFunc("/message", MessageHandler)
	err := http.ListenAndServeTLS(":6443", "server.crt", "server.key", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
