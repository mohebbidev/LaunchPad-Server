package handlers

import (
	"context"
	"fmt"
	"golaunch/internal/application"
	"net/http"
)

type RunHandler struct {
	RunUseCase *application.RunProjectUseCase
}

func NewRunHandler(uc *application.RunProjectUseCase) *RunHandler {
	return &RunHandler{RunUseCase: uc}
}

func (h *RunHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// if r.Method != http.MethodPost {
	// 	http.Error(w, "POST only", http.StatusMethodNotAllowed)
	// 	return
	// }

	projectID := r.PathValue("projectID") // Go 1.22+ stdlib routing
	if projectID == "" {
		http.Error(w, "missing projectID", http.StatusBadRequest)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// use request context so SSE disconnect cancels the stream
	logCh, err := h.RunUseCase.Execute(r.Context(), projectID)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	for line := range logCh {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", line.Stream, line.Text)
		flusher.Flush()
	}

	fmt.Fprintf(w, "event: done\ndata: process finished\n\n")
	flusher.Flush()
}
