package health

import (
	"encoding/json"
	"net/http"
)

type Tools interface {
	Ping() string
}

type listTools struct {
	name  string
	tools Tools
}

type HealthEndpoint struct {
	tools map[string]Tools
}

func (h *HealthEndpoint) AddHealthEndpoint(
	list ...listTools,
) {
	for i := 0; i < len(list); i++ {
		h.tools[list[i].name] = list[i].tools
	}
}

func (h HealthEndpoint) CheckHealthConnection(
	rw http.ResponseWriter,
	r *http.Request,
) {
	result := make(map[string]string)
	result["status"] = "UP"

	for keys := range h.tools {
		if h.tools[keys] == nil {
			result[keys] = "DOWN"
			continue
		}

		result[keys] = h.tools[keys].Ping()
	}

	response, _ := json.Marshal(result)

	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(response)
}
