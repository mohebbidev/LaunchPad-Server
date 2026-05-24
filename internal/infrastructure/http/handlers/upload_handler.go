package handlers

import (
	"context"
	"encoding/json"
	"gowsrunner/internal/application"
	"net/http"
	"strings"
)

type UploadHandler struct {
	UploadUseCase application.UploadProjectUseCase
}

func NewUplaodHandler(uploadUC *application.UploadProjectUseCase) *UploadHandler {
	return &UploadHandler{UploadUseCase: *uploadUC}
}

func (handler *UploadHandler) ServeHTTP(
	ctx context.Context, w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	// max upload size
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)

	// file validation
	file, hdr, err := r.FormFile("file")
	if err != nil {
		if err.Error() == "http: too large body" {
			http.Error(w, "file is too large (max 20MB)", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "missing file field (multipart form: file)", http.StatusBadRequest)
		}
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(hdr.Filename), ".zip") {
		http.Error(w, "only .zip supported", http.StatusBadRequest)
		return
	}

	uploadResult, err := handler.UploadUseCase.Execute(ctx, application.UploadInput{
		Filename: hdr.Filename,
		File: file,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(uploadResult)
	
}
