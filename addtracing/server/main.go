package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

type ServerInfo struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

var config map[string]ServerInfo

func main() {
	yamlFile, err := os.ReadFile("./servers.yaml")
	if err != nil {
		log.Fatal(err)
	}

	config = make(map[string]ServerInfo)
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:         ":8080",
		Handler:      enableCors(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	mux.HandleFunc("POST /calculate", handleCalculate)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

type ClientRequest struct {
	Operands string `json:"operands"`
	Operator string `json:"operator"`
}

type ApiRequest struct {
	Numbers []float64
}

type ErrorMessage struct {
	Error string `json:"error"`
}

type Response struct {
	Result float64 `json:"result"`
}

// convertToFloat convert numbers such 1,2,3 to an array of float64
func convertToFloat(numbersStr string) []float64 {
	numbers := make([]float64, 0)
	arr := strings.Split(numbersStr, ",")
	for _, number := range arr {
		n, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
		if err != nil {
			continue
		}
		numbers = append(numbers, n)
	}

	return numbers
}

func handleCalculate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "Application/json")
	var request *ClientRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	numbers := convertToFloat(request.Operands)
	apiRequest := ApiRequest{Numbers: numbers}

	serverInfo, exists := config[request.Operator]
	if !exists {
		sendError(w, "Wrong operator", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("http://%v:%v/calculate", serverInfo.Host, serverInfo.Port)
	b, err := json.Marshal(apiRequest)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sendError(w, "Wrong status code", resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, resp.Body); err != nil {
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := &ErrorMessage{Error: message}
	send(w, response)
}

func send(w http.ResponseWriter, response any) {
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
