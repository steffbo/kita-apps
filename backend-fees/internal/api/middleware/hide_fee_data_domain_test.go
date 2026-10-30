package middleware

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/knirpsenstadt/kita-apps/backend-fees/internal/domain"
)

// feeLike matches JSON field names that look like fee, income or bank data.
var feeLike = regexp.MustCompile(`(?i)fee|income|einstuf|amount|balance|iban|price|contribution`)

// TestFeeFieldsCoverDomainPayloads fails when a fee- or income-looking field is
// added to a payload that LEITUNG can read without also being listed in
// FeeFields. Fields that only look like fee data but are not (so far none)
// go into allowedFeeLike below with a reason.
func TestFeeFieldsCoverDomainPayloads(t *testing.T) {
	allowedFeeLike := map[string]string{}

	for _, v := range []interface{}{
		domain.Child{}, domain.Parent{}, domain.Household{}, domain.Member{},
		domain.ChildCareHoursHistory{}, domain.ChildLegalHoursHistory{},
	} {
		typ := reflect.TypeOf(v)
		for i := 0; i < typ.NumField(); i++ {
			name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
			if name == "" || name == "-" || !feeLike.MatchString(name) {
				continue
			}
			if _, hidden := FeeFields[name]; hidden {
				continue
			}
			if _, ok := allowedFeeLike[name]; ok {
				continue
			}
			t.Errorf("%s.%s (json %q) looks like fee data but is not in FeeFields; hide it or allow it with a reason",
				typ.Name(), typ.Field(i).Name, name)
		}
	}
}
