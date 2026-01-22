package handler

import (
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/api/response"
	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/service"
)

// ChildImportHandler handles child import requests
type ChildImportHandler struct {
	importService *service.ChildImportService
}

// NewChildImportHandler creates a new child import handler
func NewChildImportHandler(importService *service.ChildImportService) *ChildImportHandler {
	return &ChildImportHandler{importService: importService}
}

// Parse handles the initial CSV parsing request
// POST /children/import/parse
func (h *ChildImportHandler) Parse(w http.ResponseWriter, r *http.Request) {
	// Max 10MB file
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		response.BadRequest(w, "Datei zu groß oder ungültiges Format")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "Keine Datei hochgeladen")
		return
	}
	defer file.Close()

	result, err := h.importService.ParseCSV(file)
	if err != nil {
		response.InternalError(w, "Fehler beim Parsen der CSV: "+err.Error())
		return
	}

	response.Success(w, result)
}

// PreviewRequest is the request body for preview
type PreviewRequest struct {
	FileContent string         `json:"fileContent"` // Base64 encoded CSV content
	Separator   string         `json:"separator"`
	Mapping     map[string]int `json:"mapping"` // systemField -> csvColumnIndex
	SkipHeader  bool           `json:"skipHeader"`
}

// Preview handles the preview request with mapping applied
// POST /children/import/preview
func (h *ChildImportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültiges Request-Format")
		return
	}

	if req.FileContent == "" {
		response.BadRequest(w, "Keine Datei-Daten")
		return
	}

	if len(req.Mapping) == 0 {
		response.BadRequest(w, "Keine Feld-Zuordnung")
		return
	}

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(req.FileContent)
	if err != nil {
		response.BadRequest(w, "Ungültige Datei-Kodierung")
		return
	}

	// Parse CSV with specified separator
	sep := ';'
	if req.Separator != "" {
		sep = rune(req.Separator[0])
	}

	csvReader := csv.NewReader(strings.NewReader(string(content)))
	csvReader.Comma = sep
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1

	var rows [][]string
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip malformed rows
		}
		rows = append(rows, record)
	}

	// Skip header if requested
	dataRows := rows
	if req.SkipHeader && len(rows) > 0 {
		dataRows = rows[1:]
	}

	// Generate preview
	result, err := h.importService.Preview(r.Context(), dataRows, req.Mapping)
	if err != nil {
		response.InternalError(w, "Fehler bei der Vorschau: "+err.Error())
		return
	}

	response.Success(w, result)
}

// Execute handles the final import execution
// POST /children/import/execute
func (h *ChildImportHandler) Execute(w http.ResponseWriter, r *http.Request) {
	var req service.ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Ungültiges Request-Format")
		return
	}

	if len(req.Rows) == 0 {
		response.BadRequest(w, "Keine Daten zum Importieren")
		return
	}

	result, err := h.importService.Execute(r.Context(), &req)
	if err != nil {
		response.InternalError(w, "Fehler beim Import: "+err.Error())
		return
	}

	response.Success(w, result)
}
