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
func GetPagination(r *http.Request) Pagination {
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
