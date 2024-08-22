package handlehttp

import (
	"encoding/json"
	"log"
	"net/http"
)

func activateJsonResponse(w http.ResponseWriter) {
	w.Header().Set("content-type", Json.String())
}

func ToJson(w http.ResponseWriter, data any) {
	resp, err := json.Marshal(data)

	if err != nil {
		log.Println("Error while creating json response:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	activateJsonResponse(w)
	_, err = w.Write(resp)

	if err != nil {
		log.Println("Error writing response", err)
	}
}

func JsonMapper() ResponseMapper {
	return ToJson
}
