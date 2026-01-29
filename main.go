package main

import (
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
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

	// Function calling endpoint
	http.HandleFunc("/api/chat", handleChatWithFunctions)

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
	url := "https://generativelanguage.googleapis.com/v1/models/gemini-2.0-flash-latest:generateContent?key=" + apiKey

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
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	// Log response for debugging
	log.Printf("Gemini API Status: %d, Response: %s", resp.StatusCode, string(body))

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

// Function Calling with Gemini

type FunctionTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

var availableFunctions = []FunctionTool{
	{
		Name:        "calculate",
		Description: "Performs mathematical operations. Supports: add, subtract, multiply, divide, sqrt, power",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "The mathematical operation: add, subtract, multiply, divide, sqrt, power",
				},
				"a": map[string]interface{}{
					"type":        "number",
					"description": "First operand",
				},
				"b": map[string]interface{}{
					"type":        "number",
					"description": "Second operand (required for: add, subtract, multiply, divide, power)",
				},
			},
			"required": []string{"operation", "a"},
		},
	},
	{
		Name:        "get_current_time",
		Description: "Gets the current date and time in a specified timezone. Returns formatted time.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"timezone": map[string]interface{}{
					"type":        "string",
					"description": "Timezone (e.g., 'UTC', 'America/New_York', 'Europe/London'). Defaults to UTC.",
				},
			},
		},
	},
	{
		Name:        "validate_email",
		Description: "Validates if an email address has a proper format.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"email": map[string]interface{}{
					"type":        "string",
					"description": "The email address to validate",
				},
			},
			"required": []string{"email"},
		},
	},
	{
		Name:        "text_length_analysis",
		Description: "Analyzes text length, word count, and character statistics.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type":        "string",
					"description": "The text to analyze",
				},
			},
			"required": []string{"text"},
		},
	},
}

func handleChatWithFunctions(w http.ResponseWriter, r *http.Request) {
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

	// Call Gemini with basic text generation
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

func callGeminiWithFunctions(message string, apiKey string) (string, error) {
	// For now, use the basic API without function calling
	return callGeminiAPI(message, apiKey)
}

func executeFunctionSafely(name string, args map[string]interface{}) string {
	switch name {
	case "calculate":
		return executeCalculate(args)
	case "get_current_time":
		return executeGetCurrentTime(args)
	case "validate_email":
		return executeValidateEmail(args)
	case "text_length_analysis":
		return executeTextAnalysis(args)
	default:
		return `{"error": "Unknown function: ` + name + `"}`
	}
}

func executeCalculate(args map[string]interface{}) string {
	operation, ok := args["operation"].(string)
	if !ok {
		return `{"error": "operation parameter must be a string"}`
	}

	a, ok := args["a"].(float64)
	if !ok {
		return `{"error": "parameter 'a' must be a number"}`
	}

	var result float64

	switch operation {
	case "add":
		b, _ := args["b"].(float64)
		result = a + b
	case "subtract":
		b, _ := args["b"].(float64)
		result = a - b
	case "multiply":
		b, _ := args["b"].(float64)
		result = a * b
	case "divide":
		b, _ := args["b"].(float64)
		if b == 0 {
			return `{"error": "Cannot divide by zero"}`
		}
		result = a / b
	case "sqrt":
		if a < 0 {
			return `{"error": "Cannot calculate square root of negative number"}`
		}
		result = math.Sqrt(a)
	case "power":
		b, _ := args["b"].(float64)
		result = math.Pow(a, b)
	default:
		return `{"error": "Unknown operation: ` + operation + `"}`
	}

	response := map[string]interface{}{
		"operation": operation,
		"a":         a,
		"result":    result,
	}

	jsonResp, _ := json.Marshal(response)
	return string(jsonResp)
}

func executeGetCurrentTime(args map[string]interface{}) string {
	timezone := "UTC"
	if tz, ok := args["timezone"].(string); ok {
		timezone = tz
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return `{"error": "Invalid timezone: ` + timezone + `"}`
	}

	currentTime := time.Now().In(loc)

	response := map[string]interface{}{
		"timezone": timezone,
		"time":     currentTime.Format("2006-01-02 15:04:05 MST"),
		"iso8601":  currentTime.Format(time.RFC3339),
	}

	jsonResp, _ := json.Marshal(response)
	return string(jsonResp)
}

func executeValidateEmail(args map[string]interface{}) string {
	email, ok := args["email"].(string)
	if !ok {
		return `{"error": "email parameter must be a string"}`
	}

	// Basic email validation
	isValid := strings.Contains(email, "@") && strings.Contains(email, ".") && len(email) > 5
	
	response := map[string]interface{}{
		"email":   email,
		"valid":   isValid,
		"message": map[bool]string{true: "Email format is valid", false: "Email format is invalid"}[isValid],
	}

	jsonResp, _ := json.Marshal(response)
	return string(jsonResp)
}

func executeTextAnalysis(args map[string]interface{}) string {
	text, ok := args["text"].(string)
	if !ok {
		return `{"error": "text parameter must be a string"}`
	}

	words := strings.Fields(text)
	
	response := map[string]interface{}{
		"text_length":      len(text),
		"word_count":       len(words),
		"character_count":  len([]rune(text)),
		"sentence_count":   strings.Count(text, ".") + strings.Count(text, "!") + strings.Count(text, "?"),
		"average_word_len": float64(len(text)) / float64(len(words)),
	}

	jsonResp, _ := json.Marshal(response)
	return string(jsonResp)
}