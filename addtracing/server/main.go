package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:         ":8080",
		Handler:      enableCors(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	mux.HandleFunc("POST /calculate", calculate)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}

type Request struct {
	Operands string `json:"operands"`
	Operator string `json:"operator"`
}

type ErrorMessage struct {
	Error string `json:"error"`
}

type Response struct {
	Result float64
}

type Math interface {
	Calculate(numbersStr string) float64
}

type baseMathOperator struct {
}

func (s *baseMathOperator) CalculateGeneric(numbersStr string, f func(a, b float64) float64) float64 {
	result := float64(0)
	numbers := strings.Split(numbersStr, ",")
	for _, number := range numbers {
		n, err := strconv.ParseFloat(number, 64)
		if err == nil {
			result = f(result, n)
		}
	}

	return result
}

type sumMathOeprator struct{}
type multMathOperator struct{}

func doMath(numbersStr string, f func(a, b float64) float64) float64 {
	result := float64(0)
	numbers := strings.Split(numbersStr, ",")
	for idx, number := range numbers {
		n, err := strconv.ParseFloat(number, 64)
		if err == nil {
			if idx == 0 {
				result = n
				continue
			}
			result = f(result, n)
		}
	}

	return result
}

func (s *sumMathOeprator) Calculate(numbersStr string) float64 {
	return doMath(numbersStr, func(a, b float64) float64 {
		return a + b
	})
}

func (m *multMathOperator) Calculate(numbersStr string) float64 {
	return doMath(numbersStr, func(a, b float64) float64 {
		return a * b
	})
}

func GetMathOperator(operator string) Math {
	switch operator {
	case "sum":
		return &sumMathOeprator{}
	case "mult":
		return &multMathOperator{}
	default:
		return nil
	}

}

func calculate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "Application/json")
	var request *Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
	}

	math := GetMathOperator(request.Operator)
	result := math.Calculate(request.Operands)

	response := &Response{Result: result}
	w.WriteHeader(http.StatusOK)
	send(w, response)
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
