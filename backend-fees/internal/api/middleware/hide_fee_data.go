package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// FeeFields are the JSON keys on child, parent, household and member payloads
// that carry fee or income information. HideFeeData removes them from
// responses and request bodies of the restricted roles.
//
// New fee- or income-related fields on those payloads must be added here;
// TestFeeFieldsCoverDomainPayloads fails when a suspicious field is missing.
var FeeFields = map[string]struct{}{
	"annualHouseholdIncome": {},
	"incomeStatus":          {},
	"childrenCountForFees":  {},
	"openFeesCount":         {},
	// Result of the automatic Einstufung/fee sync after a child update.
	"einstufungSync": {},
}

// feeQueryParams filter the child list by fee state or by warnings that
// include missing income data; they would leak what the fields above hide.
var feeQueryParams = []string{"hasOpenFees", "hasWarnings"}

// HideFeeData keeps fee and income information away from the given roles
// (e.g. LEITUNG). It runs after AuthMiddleware and is a no-op for every
// other role.
//
//   - Responses: FeeFields are removed from JSON bodies at any depth, so
//     nested households and parents are covered too.
//   - Requests: FeeFields are dropped from JSON bodies before the handler runs,
//     so such a role can neither read nor change them (absent means unchanged).
//   - Query: the fee-related list filters are ignored.
func HideFeeData(roles ...string) func(http.Handler) http.Handler {
	restricted := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		restricted[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userCtx := GetUserFromContext(r)
			if userCtx == nil {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := restricted[userCtx.Role]; !ok {
				next.ServeHTTP(w, r)
				return
			}

			stripQuery(r)
			if err := stripRequestBody(r); err != nil {
				http.Error(w, `{"error":"Bad Request","message":"invalid request body","code":400}`,
					http.StatusBadRequest)
				return
			}

			buf := &bufferedResponse{header: w.Header(), status: http.StatusOK}
			next.ServeHTTP(buf, r)
			buf.flushTo(w)
		})
	}
}

func stripQuery(r *http.Request) {
	q := r.URL.Query()
	changed := false
	for _, name := range feeQueryParams {
		if q.Has(name) {
			q.Del(name)
			changed = true
		}
	}
	if changed {
		r.URL.RawQuery = q.Encode()
	}
}

func stripRequestBody(r *http.Request) error {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
		return nil
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !isJSON(ct) {
		return nil
	}

	raw, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		return err
	}
	out, changed := stripFeeFields(raw)
	if !changed {
		out = raw
	}
	r.Body = io.NopCloser(bytes.NewReader(out))
	r.ContentLength = int64(len(out))
	r.Header.Set("Content-Length", strconv.Itoa(len(out)))
	return nil
}

func isJSON(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "application/json")
}

// stripFeeFields removes FeeFields from a JSON document. It reports false and
// leaves the input to the caller when the document is not valid JSON or has
// nothing to remove.
func stripFeeFields(raw []byte) ([]byte, bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // keep numbers byte-exact
	var doc interface{}
	if err := dec.Decode(&doc); err != nil {
		return raw, false
	}
	if !removeFeeFields(doc) {
		return raw, false
	}
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		return raw, false
	}
	return bytes.TrimRight(out.Bytes(), "\n"), true
}

// removeFeeFields deletes FeeFields recursively and reports whether it
// removed anything.
func removeFeeFields(v interface{}) bool {
	changed := false
	switch t := v.(type) {
	case map[string]interface{}:
		for key, val := range t {
			if _, hidden := FeeFields[key]; hidden {
				delete(t, key)
				changed = true
				continue
			}
			if removeFeeFields(val) {
				changed = true
			}
		}
	case []interface{}:
		for _, val := range t {
			if removeFeeFields(val) {
				changed = true
			}
		}
	}
	return changed
}

// bufferedResponse holds a handler's response so that its JSON body can be
// filtered before anything reaches the client.
type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (b *bufferedResponse) Header() http.Header { return b.header }

func (b *bufferedResponse) WriteHeader(status int) { b.status = status }

func (b *bufferedResponse) Write(p []byte) (int, error) { return b.body.Write(p) }

func (b *bufferedResponse) flushTo(w http.ResponseWriter) {
	body := b.body.Bytes()
	if isJSON(b.header.Get("Content-Type")) && len(body) > 0 {
		if out, changed := stripFeeFields(body); changed {
			body = append(out, '\n')
		}
	}
	b.header.Del("Content-Length")
	w.WriteHeader(b.status)
	if len(body) > 0 {
		_, _ = w.Write(body)
	}
}
