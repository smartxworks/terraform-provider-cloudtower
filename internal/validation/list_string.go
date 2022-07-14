package validation

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ListStringInSlice(valid []string, ignoreCase bool) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (warnings []string, errors []error) {
		v, ok := i.([]interface{})
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %q to be List", k))
			return warnings, errors
		}

		for _, rawVal := range v {
			val, ok := rawVal.(string)
			if !ok {
				errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
				continue
			}
			for _, str := range valid {
				if val == str || (ignoreCase && strings.EqualFold(val, str)) {
					break
				}
				errors = append(errors, fmt.Errorf("%s must be one of %v, got %s", k, valid, val))
			}
		}
		return warnings, errors
	}
}
