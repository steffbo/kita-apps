package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serve runs h behind HideFeeData(LEITUNG) as the given role.
func serve(t *testing.T, role, method, target, contentType, body string, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req = req.WithContext(context.WithValue(req.Context(), UserContextKey, &UserContext{Role: role}))
	rec := httptest.NewRecorder()
	HideFeeData("LEITUNG")(h).ServeHTTP(rec, req)
	return rec
}

func jsonHandler(status int, payload string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(payload))
	}
}

const childPayload = `{"data":[{"id":"c1","firstName":"Emma","openFeesCount":3,` +
	`"household":{"id":"h1","name":"Familie Müller","annualHouseholdIncome":65000.5,` +
	`"incomeStatus":"PROVIDED","childrenCountForFees":2,"membershipAssignmentStatus":"CONFIRMED"},` +
	`"parents":[{"id":"p1","firstName":"Thomas","annualHouseholdIncome":1,"incomeStatus":"PENDING",` +
	`"household":{"id":"h1","incomeStatus":"PROVIDED"}}]}],"total":1}`

func TestHideFeeData_RemovesFeeFieldsFromResponse(t *testing.T) {
	rec := serve(t, "LEITUNG", http.MethodGet, "/children", "", "", jsonHandler(http.StatusOK, childPayload))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for key := range FeeFields {
		if strings.Contains(body, `"`+key+`"`) {
			t.Errorf("response still contains %q: %s", key, body)
		}
	}
	for _, keep := range []string{`"firstName":"Emma"`, `"name":"Familie Müller"`, `"membershipAssignmentStatus":"CONFIRMED"`, `"total":1`} {
		if !strings.Contains(body, keep) {
			t.Errorf("response lost %s: %s", keep, body)
		}
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type %q", got)
	}
}

func TestHideFeeData_OtherRolesSeeEverything(t *testing.T) {
	for _, role := range []string{"ADMIN", "USER", "PARENT_WORK"} {
		rec := serve(t, role, http.MethodGet, "/children", "", "", jsonHandler(http.StatusOK, childPayload))
		if rec.Body.String() != childPayload {
			t.Errorf("%s: body changed: %s", role, rec.Body.String())
		}
	}
}

func TestHideFeeData_KeepsStatusAndNoContent(t *testing.T) {
	created := serve(t, "LEITUNG", http.MethodPost, "/parents", "application/json", `{}`,
		jsonHandler(http.StatusCreated, `{"id":"p1","incomeStatus":"PENDING"}`))
	if created.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201", created.Code)
	}
	if strings.Contains(created.Body.String(), "incomeStatus") {
		t.Errorf("body: %s", created.Body.String())
	}

	deleted := serve(t, "LEITUNG", http.MethodDelete, "/parents/1", "", "",
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("status %d body %q, want empty 204", deleted.Code, deleted.Body.String())
	}
}

func TestHideFeeData_ErrorBodiesAndNonJSONPassThrough(t *testing.T) {
	errRec := serve(t, "LEITUNG", http.MethodGet, "/children/1", "", "",
		jsonHandler(http.StatusNotFound, `{"error":"Not Found","code":404}`))
	if errRec.Code != http.StatusNotFound || errRec.Body.String() != `{"error":"Not Found","code":404}` {
		t.Fatalf("error response changed: %d %s", errRec.Code, errRec.Body.String())
	}

	csv := serve(t, "LEITUNG", http.MethodGet, "/members/export", "", "",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/csv")
			_, _ = w.Write([]byte(`a,"incomeStatus"` + "\n"))
		})
	if csv.Body.String() != `a,"incomeStatus"`+"\n" {
		t.Fatalf("non-JSON body changed: %q", csv.Body.String())
	}
}

func TestHideFeeData_DropsFeeFieldsFromRequestBody(t *testing.T) {
	var got map[string]interface{}
	var length int64
	handler := func(w http.ResponseWriter, r *http.Request) {
		length = r.ContentLength
		raw, _ := io.ReadAll(r.Body)
		if int64(len(raw)) != length {
			t.Errorf("ContentLength %d, body has %d bytes", length, len(raw))
		}
		_ = json.Unmarshal(raw, &got)
		w.WriteHeader(http.StatusNoContent)
	}
	body := `{"name":"Familie <A&B>","annualHouseholdIncome":99999.99,"incomeStatus":"PROVIDED",` +
		`"childrenCountForFees":3,"membershipParentId":"p1","big":12345678901234567890}`

	rec := serve(t, "LEITUNG", http.MethodPut, "/households/1", "application/json", body, handler)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
	for key := range FeeFields {
		if _, ok := got[key]; ok {
			t.Errorf("handler received %q", key)
		}
	}
	if got["name"] != "Familie <A&B>" || got["membershipParentId"] != "p1" {
		t.Errorf("other fields changed: %v", got)
	}
	if got["big"] != float64(12345678901234567890) {
		t.Errorf("number changed: %v", got["big"])
	}
}

func TestHideFeeData_RequestBodyOfOtherRolesUntouched(t *testing.T) {
	body := `{"annualHouseholdIncome":1}`
	var seen string
	serve(t, "USER", http.MethodPut, "/households/1", "application/json", body,
		func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			seen = string(raw)
		})
	if seen != body {
		t.Fatalf("handler saw %q", seen)
	}
}

func TestHideFeeData_InvalidOrNonJSONRequestReachesHandler(t *testing.T) {
	for name, tc := range map[string]struct{ contentType, body string }{
		"invalid json": {"application/json", `{"incomeStatus":`},
		"multipart":    {"multipart/form-data; boundary=x", `--x incomeStatus`},
	} {
		var seen string
		serve(t, "LEITUNG", http.MethodPost, "/x", tc.contentType, tc.body,
			func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				seen = string(raw)
			})
		if seen != tc.body {
			t.Errorf("%s: handler saw %q, want %q", name, seen, tc.body)
		}
	}
}

func TestHideFeeData_IgnoresFeeListFilters(t *testing.T) {
	var query string
	serve(t, "LEITUNG", http.MethodGet, "/children?hasOpenFees=true&hasWarnings=true&search=Emma&page=2", "", "",
		func(w http.ResponseWriter, r *http.Request) { query = r.URL.RawQuery })
	if strings.Contains(query, "hasOpenFees") || strings.Contains(query, "hasWarnings") {
		t.Errorf("fee filters survived: %s", query)
	}
	if !strings.Contains(query, "search=Emma") || !strings.Contains(query, "page=2") {
		t.Errorf("other params lost: %s", query)
	}

	serve(t, "USER", http.MethodGet, "/children?hasOpenFees=true", "", "",
		func(w http.ResponseWriter, r *http.Request) { query = r.URL.RawQuery })
	if query != "hasOpenFees=true" {
		t.Errorf("USER query changed: %s", query)
	}
}

func TestHideFeeData_UnauthenticatedPassesThrough(t *testing.T) {
	rec := httptest.NewRecorder()
	HideFeeData("LEITUNG")(jsonHandler(http.StatusOK, `{"incomeStatus":"x"}`)).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Body.String() != `{"incomeStatus":"x"}` {
		t.Fatalf("body %s", rec.Body.String())
	}
}
