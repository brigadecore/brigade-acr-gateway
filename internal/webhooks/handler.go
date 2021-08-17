package webhooks

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

// handler is an implementation of the http.Handler interface that can handle
// webhooks (events) from ACR by delegating to a transport-agnostic Service
// interface.
type handler struct {
	service Service
}

// handler is an implementation of the http.Handler interface that can handle
// webhooks (events) from ACR by delegating to a transport-agnostic Service
// interface.
func NewHandler(service Service) (http.Handler, error) {
	return &handler{
		service: service,
	}, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Set("Content-Type", "application/json")

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status": "failed"}`)) // nolint: errcheck
		return
	}

	event := Event{}
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status": "failed"}`)) // nolint: errcheck
		return
	}

	if err := h.service.Handle(r.Context(), event, payload); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "internal server error"}`)) // nolint: errcheck
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`)) // nolint: errcheck
}
