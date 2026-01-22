package request

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// DecodeJSON decodes a JSON request body into the given struct.
func DecodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Pagination holds pagination parameters from query strings.
type Pagination struct {
	Page    int
	PerPage int
	Offset  int
}

// GetPagination extracts pagination parameters from the request.
// Supports two conventions:
// 1. offset/limit style: ?offset=20&limit=10
// 2. page/perPage style: ?page=3&perPage=10
// If offset/limit are provided, they take precedence.
func GetPagination(r *http.Request) Pagination {
	// Check for offset/limit style first
	offsetParam := r.URL.Query().Get("offset")
	limitParam := r.URL.Query().Get("limit")

	if offsetParam != "" || limitParam != "" {
		// Use offset/limit style
		offset := getQueryInt(r, "offset", 0)
		limit := getQueryInt(r, "limit", 20)

		if offset < 0 {
			offset = 0
		}
		if limit < 1 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		// Calculate page from offset for response
		page := (offset / limit) + 1

		return Pagination{
			Page:    page,
			PerPage: limit,
			Offset:  offset,
		}
	}

	// Fall back to page/perPage style
	page := getQueryInt(r, "page", 1)
	perPage := getQueryInt(r, "perPage", 20)

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	return Pagination{
		Page:    page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
	}
}

// GetQueryString returns a query string parameter with a default value.
func GetQueryString(r *http.Request, key, defaultValue string) string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetQueryBool returns a boolean query parameter.
func GetQueryBool(r *http.Request, key string) *bool {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil
	}
	b := value == "true" || value == "1"
	return &b
}

// GetQueryInt returns an integer query parameter with a default value.
func getQueryInt(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return i
}

// GetQueryIntOptional returns an optional integer query parameter.
func GetQueryIntOptional(r *http.Request, key string) *int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return nil
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &i
}
