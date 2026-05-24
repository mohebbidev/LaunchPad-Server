package utils

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
)

func NewID() string {
	return uuid.New().String()
}

func GetEnv(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return fallback
}



func OpenJSON[T any](fileURL string) (T, error) {
	var null T
	file, err := os.Open(fileURL)
	if err != nil {
		return null, err
	}
	defer file.Close()

	var content T
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&content)
	if err != nil {
		return null, err
	}
	return content, nil
}