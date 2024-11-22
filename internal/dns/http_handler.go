package dns

import (
	"context"
	"io"
	"log"
	"net/http"
)

type HTTPResponseWriter struct {
	w http.ResponseWriter
}

func (w *HTTPResponseWriter) WriteToResponse(data []byte) error {
	w.w.Header().Set("Content-Type", "application/dns-message")
	w.w.WriteHeader(http.StatusOK)
	_, err := w.w.Write(data)
	return err
}

func (h *DNSHandler) HttpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength < 12 {
		http.Error(w, "Query is too short", http.StatusBadRequest)
		return
	}

	dnsQuery, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request: %s", err.Error())
		http.Error(w, "Failed to read request", http.StatusInternalServerError)
		return
	}

	writer := &HTTPResponseWriter{w: w}
	h.HandleDNSQuery(context.Background(), dnsQuery, writer)
}
