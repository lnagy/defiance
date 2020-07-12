package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	serverURL := os.Args[1]
	playerKey := os.Args[2]

	fmt.Printf("ServerUrl: %s; PlayerKey: %s\n", serverURL, playerKey)

	res, err := http.Get(fmt.Sprintf("%s?playerKey=%s", serverURL, playerKey))
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Fatalf("Failed: got status %d, want %d", res.StatusCode, http.StatusOK)
	}
}
