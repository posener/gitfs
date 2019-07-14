// An example locally serves files from github.com/golang/go/doc.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/posener/gitfs"
)

func main() {
	ctx := context.Background()
	fs, err := gitfs.New(ctx, "github.com/golang/go/doc")
	if err != nil {
		log.Fatalf("Failed initializing git filesystem: %s.", err)
	}
	http.Handle("/", http.RedirectHandler("/doc/", http.StatusMovedPermanently))
	http.Handle("/doc/", http.StripPrefix("/doc/", http.FileServer(fs)))
	log.Printf("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
