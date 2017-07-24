package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("The request actually got here")

		w.Write([]byte("You got here"))
	})

	fmt.Println("Listening on port 8000")
	http.ListenAndServe(":8000", mux)
}
