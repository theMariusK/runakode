package handlers

import (
	"fmt"
	"net/http"
	"log"
	"encoding/json"
	"github.com/theMariusK/runakode/api/config"
	"slices"
)

type RunRequest struct {
	Language string `json:"language"`
	SourceCode string `json:"source_code"`
}

type RunResponse struct {
	id int `json:"id"`
	result string `json:"result"`
	exitCode int `json:"exit_code"`
}

func Api(conf *config.Config) (http.HandlerFunc) {
	return func(w http.ResponseWriter, r *http.Request) {
	        log.Printf("Got a %s request from %s\n", r.Method, r.RemoteAddr)
	        if r.Method != http.MethodPost {
		        http.Error(w, "Wrong method!", http.StatusMethodNotAllowed)
		        return
	        }

	        var request RunRequest
	        err := json.NewDecoder(r.Body).Decode(&request)
	        if err != nil {
	 	        log.Println(err.Error())
		        http.Error(w, "Cant parse the request!", http.StatusBadRequest)
		        return
	        }

		if ! slices.Contains(conf.SupportedLanguages, request.Language) {
			http.Error(w, "Unsupported language!", http.StatusBadRequest)
			return
		}
	        fmt.Fprintf(w, request.SourceCode)
        }
}
