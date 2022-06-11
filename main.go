package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	// TODO: Read configs from yaml file
	// TODO: Validate configs
	urlToRequest := "http://discuss.pytorch.kr"
	methodForReq := "GET"
	timeoutInSec := 5

	// Set global configs
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeoutInSec))
	defer cancel()

	// Generate Request from configs
	req, err := http.NewRequestWithContext(ctx, methodForReq, urlToRequest, nil)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			log.Fatal(err)
			os.Exit(1)
		}
	}
	defer resp.Body.Close()

	// Parse response
	log.Println(resp.Header)

	// Save response to log file

	// Visualize logs

}
