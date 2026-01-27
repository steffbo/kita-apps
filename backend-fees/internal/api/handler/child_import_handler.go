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

// ParseCSVResponse represents the result of parsing a CSV file
// @Description CSV parsing result with headers and sample data
type ParseCSVResponse struct {
	Headers     []string   `json:"headers" example:"Vorname,Nachname,Geburtsdatum,Gruppe"`
	SampleRows  [][]string `json:"sampleRows"`
	TotalRows   int        `json:"totalRows" example:"50"`
	Separator   string     `json:"separator" example:";"`
	FileContent string     `json:"fileContent"` // Base64 encoded for re-use in preview
}

// ChildPreviewRow represents a preview row for child import
// @Description Single child preview with validation status
type ChildPreviewRow struct {
	RowNumber   int      `json:"rowNumber" example:"1"`
	FirstName   string   `json:"firstName" example:"Max"`
	LastName    string   `json:"lastName" example:"Mustermann"`
	BirthDate   *string  `json:"birthDate,omitempty" example:"2020-05-15"`
	GroupName   *string  `json:"groupName,omitempty" example:"Schmetterlinge"`
	ParentName  *string  `json:"parentName,omitempty" example:"Hans Mustermann"`
	ParentEmail *string  `json:"parentEmail,omitempty" example:"hans@example.com"`
	IsValid     bool     `json:"isValid" example:"true"`
	Errors      []string `json:"errors,omitempty" example:"Ungültiges Geburtsdatum"`
	Warnings    []string `json:"warnings,omitempty" example:"Gruppe nicht gefunden"`
	Action      string   `json:"action" example:"create" enums:"create,update,skip"`
	ExistingID  *string  `json:"existingId,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// PreviewResponse represents the result of previewing a child import
// @Description Import preview with validation results
type PreviewResponse struct {
	Rows      []ChildPreviewRow `json:"rows"`
	TotalRows int               `json:"totalRows" example:"50"`
	ValidRows int               `json:"validRows" example:"45"`
	ToCreate  int               `json:"toCreate" example:"30"`
	ToUpdate  int               `json:"toUpdate" example:"15"`
	ToSkip    int               `json:"toSkip" example:"5"`
	HasErrors bool              `json:"hasErrors" example:"false"`
}

// ExecuteImportResponse represents the result of executing a child import
// @Description Import execution result
type ExecuteImportResponse struct {
	Created int      `json:"created" example:"30"`
	Updated int      `json:"updated" example:"15"`
	Skipped int      `json:"skipped" example:"5"`
	Errors  []string `json:"errors,omitempty"`
}

// NewChildImportHandler creates a new child import handler
func NewChildImportHandler(importService *service.ChildImportService) *ChildImportHandler {
	return &ChildImportHandler{importService: importService}
}

// Parse handles the initial CSV parsing request
// POST /children/import/parse
// @Summary Parse a CSV file for child import
// @Description Upload and parse a CSV file to extract headers and sample data for mapping
// @Tags Children Import
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "CSV file (max 10MB)"
// @Success 200 {object} ParseCSVResponse "Parsed CSV structure"
// @Failure 400 {object} response.ErrorBody "File too large or invalid format"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/import/parse [post]
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
// @Description Request body for generating import preview
type PreviewRequest struct {
	FileContent string         `json:"fileContent"` // Base64 encoded CSV content
	Separator   string         `json:"separator" example:";"`
	Mapping     map[string]int `json:"mapping"` // systemField -> csvColumnIndex
	SkipHeader  bool           `json:"skipHeader" example:"true"`
}

// Preview handles the preview request with mapping applied
// POST /children/import/preview
// @Summary Preview child import with mapping
// @Description Apply field mapping to CSV data and generate a preview with validation
// @Tags Children Import
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param preview body PreviewRequest true "Preview configuration"
// @Success 200 {object} PreviewResponse "Import preview"
// @Failure 400 {object} response.ErrorBody "Invalid request or missing mapping"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/import/preview [post]
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
// @Summary Execute child import
// @Description Execute the import with the validated and confirmed data
// @Tags Children Import
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param execute body service.ExecuteRequest true "Import data"
// @Success 200 {object} ExecuteImportResponse "Import result"
// @Failure 400 {object} response.ErrorBody "Invalid request or no data to import"
// @Failure 401 {object} response.ErrorBody "Not authenticated"
// @Failure 500 {object} response.ErrorBody "Internal server error"
// @Router /children/import/execute [post]
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
