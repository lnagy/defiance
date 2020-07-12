package main

import (
	"app/qsort"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	fmt.Printf("%v", qsort.Sort([]int{}))
	serverURL := os.Args[1]
	playerKey := os.Args[2]

	fmt.Printf("ServerUrl: %s; PlayerKey: %s\n", serverURL, playerKey)

	res, err := http.Get(fmt.Sprintf("%s?playerKey=%s", serverURL, playerKey))
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	defer func() {
		if res := res.Body.Close(); res != nil {
			log.Printf("Error closing connection: %v", res)
		}
	}()

	if res.StatusCode != http.StatusOK {
		log.Fatalf("Failed: got status %d, want %d", res.StatusCode, http.StatusOK)
	}
}
