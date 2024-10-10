package handlehttp

import (
	"net/http"
)

type CorsConfig struct {
	AddCorsHeader bool
	CorsWhitelist string
}

func AddCorsHeader(config CorsConfig, next GetResponseMapper) GetResponseMapper {
	return func(r *http.Request) ResponseMapper {
		mapper := next(r)
		var newMapper ResponseMapper = func(w http.ResponseWriter, input MappingInput) {
			setCorsHeaders(w, config)
			mapper(w, input)
		}
		return newMapper
	}
}

func setCorsHeaders(w http.ResponseWriter, config CorsConfig) {
	if config.AddCorsHeader {
		w.Header().Set("Access-Control-Allow-Origin", config.CorsWhitelist)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
}
