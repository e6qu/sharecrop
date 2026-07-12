package db

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StringArray scans a text-array aggregation from either engine into a slice:
// Postgres renders array_agg as an array literal ("{a,b}"), while the SQLite
// dialect rewrites array_agg to json_group_array, which renders JSON ("[a,b]").
// One Scanner handles both so array_agg-backed columns (agent/org credential
// scopes, membership roles) work unchanged on both engines.
//
// This file is a policy-check boundary: sql.Scanner's Scan(any) signature is
// unavoidably weakly typed.
type StringArray []string

func (array *StringArray) Scan(src any) error {
	if src == nil {
		*array = nil
		return nil
	}

	var text string
	switch value := src.(type) {
	case string:
		text = value
	case []byte:
		text = string(value)
	default:
		return fmt.Errorf("db: cannot scan %T into StringArray", src)
	}

	text = strings.TrimSpace(text)
	if text == "" || text == "{}" || text == "[]" {
		*array = []string{}
		return nil
	}

	if strings.HasPrefix(text, "[") {
		var values []string
		if err := json.Unmarshal([]byte(text), &values); err != nil {
			return err
		}
		*array = values
		return nil
	}

	*array = parsePostgresArray(text)
	return nil
}

// parsePostgresArray parses a Postgres array literal like {a,b,"c d"}. The
// aggregated columns hold simple identifier values (scopes, roles) with nulls
// already removed, so quoting is handled but nested arrays are not.
func parsePostgresArray(text string) []string {
	text = strings.TrimPrefix(text, "{")
	text = strings.TrimSuffix(text, "}")
	if text == "" {
		return []string{}
	}

	values := make([]string, 0)
	for _, element := range strings.Split(text, ",") {
		element = strings.TrimSpace(element)
		if element == "" || element == "NULL" {
			continue
		}
		if strings.HasPrefix(element, `"`) && strings.HasSuffix(element, `"`) {
			element = strings.TrimPrefix(element, `"`)
			element = strings.TrimSuffix(element, `"`)
			element = strings.ReplaceAll(element, `\"`, `"`)
		}
		values = append(values, element)
	}
	return values
}
