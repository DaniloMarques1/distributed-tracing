package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"danilo.marques/calculate/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	yaml "gopkg.in/yaml.v3"
)

type ServerInfo struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

var config map[string]ServerInfo

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	yamlFile, err := os.ReadFile("./servers.yaml")
	if err != nil {
		log.Fatal(err)
	}

	config = make(map[string]ServerInfo)
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		log.Fatal(err)
	}

	shutdown, err := tracing.InitTracer(ctx, "calculate")
	if err != nil {
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

	handler := otelhttp.NewHandler(http.HandlerFunc(handleCalculate), "calculate")
	mux.Handle("POST /calculate", handler)

	go func() {
		log.Println("Server running on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error %v\n", err)
	}

	if err := shutdown(shutdownCtx); err != nil {
		log.Printf("Tracer shutdown error %v\n", err)
	}

	log.Println("Server exited")
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
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

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
		span.SetStatus(codes.Error, "server does not exist")
		sendError(w, "Wrong operator", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("http://%v:%v/calculate", serverInfo.Host, serverInfo.Port)
	b, err := json.Marshal(apiRequest)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	resp, err := client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		span.SetStatus(codes.Error, fmt.Sprintf("Wrong status code returned %v", resp.StatusCode))
		sendError(w, "Wrong status code", resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, resp.Body); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
