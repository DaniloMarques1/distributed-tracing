package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	http.HandleFunc("/calculate", handleFunc)

	log.Fatal(http.ListenAndServe(":3003", nil))
}

func handleFunc(w http.ResponseWriter, r *http.Request) {
	request := &Request{}
	if err := json.NewDecoder(r.Body).Decode(request); err != nil {
		errResponse := &ErrorResponse{Message: fmt.Sprintf("ERROR: %v", err.Error())}
		if err := json.NewEncoder(w).Encode(errResponse); err != nil {
			log.Fatal(err)
		}

		return
	}

	var result float64
	for idx, number := range request.Numbers {
		if idx == 0 {
			result = number
			continue
		}
		result /= number
	}

	response := &Response{Result: result}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Fatal(err)
		return
	}

	return
}
