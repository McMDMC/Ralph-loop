package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	// Strip the "static" prefix from the filesystem
	staticFS, _ := fs.Sub(staticFiles, "static")
	
	// Serve static assets via /static/ path
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Serve index.html at the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, _ := fs.ReadFile(staticFS, "index.html")
		w.Header().Set("Content-Type", "text/html")
		w.Write(content)
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	log.Printf("Orchestrator starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}