package handlehttp

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteAsJson(w http.ResponseWriter, input MappingInput) {
	var resp []byte
	var err error
	data := input.Data
	status := input.Ctx.Status

	if data != nil {
		resp, err = json.Marshal(data)

		if err != nil {
			log.Println("Error while creating json response:", err)
			status = http.StatusInternalServerError
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if resp != nil {
		_, err = w.Write(resp)

		if err != nil {
			log.Println("Error writing response", err)
		}
	}
}

var JsonMapper ResponseMapper = WriteAsJson
