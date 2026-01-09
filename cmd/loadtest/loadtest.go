//nolint:all
package main

import (
	"log"
	"net/http"
)

func main() {
	/*
		TODO:
			user signup []
			user login []
			websocket connect []
			send message []
	*/

	endpointSignup := "http://localhost:8080/account/signup"
	// endpointLogin := "localhost:8080/account/login"
	// endpointChat := "localhost:8080/chat"

	res, err := http.Get(endpointSignup)
	if err != nil {
		log.Printf("failed to send GET request to [%s]: %v", endpointSignup, err)
		return
	}
	defer res.Body.Close()
}
