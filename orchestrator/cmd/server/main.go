package main

import (
	"log"
	"net/http"
	"os"

	orchestrator "github.com/camdenclark/gcrunner/orchestrator"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/task/", orchestrator.HandleTask)
	http.HandleFunc("/", orchestrator.HandleWebhook)

	log.Printf("gcrunner listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
