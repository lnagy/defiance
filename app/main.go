package main

import (
	"app/eval"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	inputFile := flag.String("input_file", "",
		"Filename to parse expressions from.")
	evaluateId := flag.String("evaluate", "",
		"Name of the expression to evaluate.")
	flag.Parse()

	if len(*inputFile) > 0 {
		bytes, err := ioutil.ReadFile(*inputFile)
		if err != nil {
			log.Fatalln("Failed to read file: ", *inputFile, "  error: ", err)
		}
		contents := string(bytes)
		parser := eval.Parser{}
		if _, err := parser.Parse(contents); err != nil {
			log.Fatalln("Failed to parse file: ", *inputFile, "  error: ", err)
		}
		_, ioErr := fmt.Fprintf(os.Stderr, "Parse finished. Variables: %v  Nodes: %v  Recursive Definitions: %v\n",
			len(parser.Vars), parser.NodeCount, parser.RecursiveCount)
		if ioErr != nil {
			// Do nothing.
		}
		if len(*evaluateId) > 0 {
			node, ok := parser.Vars[*evaluateId]
			if !ok {
				log.Fatalf("Unknown variable: '%v'\n", *evaluateId)
			}
			reducer := parser.NewReducer(node, false)
			reducer.PrintSteps = true
			result, err := reducer.Reduce(reducer.Root)
			if err != nil {
				log.Fatalf("Failed to reduce expression '%v'. Error: %v", *evaluateId, err)
			}
			fmt.Println(result)
		}
		return
	}

	serverURL := os.Args[1]
	playerKey := os.Args[2]

	fmt.Printf("ServerUrl: %s; PlayerKey: %s\n", serverURL, playerKey)

	res, err := http.Post(serverURL, "text/plain", strings.NewReader(playerKey))
	if err != nil {
		log.Printf("Unexpected server response:\n%v", err)
		os.Exit(1)
	}
	defer func() {
		if res := res.Body.Close(); res != nil {
			log.Printf("Error closing connection: %v", res)
		}
	}()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Unexpected server response:\n%v", err)
		os.Exit(1)
	}

	if res.StatusCode != http.StatusOK {
		log.Printf("Unexpected server response:")
		log.Printf("HTTP code: %d", res.StatusCode)
		log.Printf("Response body: %s", body)
		os.Exit(2)
	}

	log.Printf("Server response: %s", body)
}
