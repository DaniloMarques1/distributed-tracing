package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"danilo.marques/distributed/sum/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Request struct {
	Numbers []float64 `json:"numbers"`
}

type Response struct {
	Result float64 `json:"result"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdown, err := tracing.InitTracer(ctx, "sum")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	server := &http.Server{Addr: ":3000", Handler: mux}
	handler := otelhttp.NewHandler(http.HandlerFunc(handleFunc), "sum/calculate")
	mux.Handle("POST /calculate", handler)

	http.HandleFunc("POST /calculate", handleFunc)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully")

	shutDownCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := server.Shutdown(shutDownCtx); err != nil {
		log.Println("Error shutting down server")
	}

	if err := shutdown(shutDownCtx); err != nil {
		log.Println("Error shutting down tracing")
	}

	log.Println("Server finished running")
}

func handleFunc(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	request := &Request{}
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		errResponse := &ErrorResponse{Message: fmt.Sprintf("ERROR: %v", err.Error())}
		if err := json.NewEncoder(w).Encode(errResponse); err != nil {
			log.Printf("%v\n", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}

		return
	}

	if len(request.Numbers) == 0 {
		span.SetStatus(codes.Error, "Numbers cannot be empty")

		response := &ErrorResponse{Message: "Numbers cannot be empty"}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return
		}

		return
	}

	result := float64(0)
	for _, number := range request.Numbers {
		result += number
	}

	response := &Response{Result: result}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	return
}
