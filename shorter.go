package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello world")
	})
	http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hi")
	})

	port := ":8080"
	fmt.Println("Server is running on port" + port)

	// Start server on port specified above
	log.Fatal(http.ListenAndServe(port, nil))
}
