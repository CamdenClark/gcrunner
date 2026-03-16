package main

import (
	"log"
	"net/http"
	"os"

	function "github.com/camdenclark/gcrunner/function"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", function.HandleWebhook)

	log.Printf("gcrunner listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
