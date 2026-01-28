package main

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
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

	// Gemini API endpoint
	http.HandleFunc("/api/ask", handleGeminiRequest)

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	log.Printf("Orchestrator starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleGeminiRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Message string `json:"message"`
	}

	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &request)

	if request.Message == "" {
		http.Error(w, "No message provided", http.StatusBadRequest)
		return
	}

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		http.Error(w, "Gemini API key not configured", http.StatusInternalServerError)
		return
	}

	// Call Gemini API
	response, err := callGeminiAPI(request.Message, geminiAPIKey)
	if err != nil {
		log.Printf("Gemini API error: %v", err)
		http.Error(w, "Error calling Gemini API", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
	})
}

func callGeminiAPI(message string, apiKey string) (string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": message,
					},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Extract text from response
	if candidates, ok := result["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							return text, nil
						}
					}
				}
			}
		}
	}

	return "Unable to generate response", nil
}