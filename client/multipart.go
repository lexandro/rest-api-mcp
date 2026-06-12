package client

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// buildMultipartBody constructs a multipart/form-data body from text fields and
// local file paths. Returns the body reader and the Content-Type header value
// (which includes the generated boundary).
func buildMultipartBody(files map[string]string, fields map[string]string) (io.Reader, string, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, "", fmt.Errorf("writing form field %s: %w", name, err)
		}
	}

	for fieldName, filePath := range files {
		if err := appendFilePart(writer, fieldName, filePath); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("finalizing multipart body: %w", err)
	}
	return &buffer, writer.FormDataContentType(), nil
}

func appendFilePart(writer *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening upload file %s: %w", filePath, err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("creating form file %s: %w", fieldName, err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("reading upload file %s: %w", filePath, err)
	}
	return nil
}
