package platform

// FatSecret error code 14 ("Missing scope"). /foods/search/v6 is Premier
// exclusive, so Basic plan keys get this error and use /foods/search/v1.
const missingScopeErrorCode = 14

func IsMissingScopeError(data any) bool {
	m, ok := data.(map[string]any)
	if !ok {
		return false
	}
	em, ok := m["error"].(map[string]any)
	if !ok {
		return false
	}
	switch c := em["code"].(type) {
	case string:
		return parseNum(c) == missingScopeErrorCode
	case float64:
		return c == missingScopeErrorCode
	}
	return false
}

var searchServingFields = []string{
	"serving_id", "serving_description", "measurement_description",
	"number_of_units", "metric_serving_amount", "metric_serving_unit",
	"is_default", "calories", "protein", "fat", "carbohydrate",
}

// trimSearchServings: /foods/search/v6 returns every serving with ~20
// micronutrients; search only needs enough to pick a food and serving.
func TrimSearchServings(data any) any {
	m, ok := data.(map[string]any)
	if !ok {
		return data
	}
	rs, ok := m["foods_search"].(map[string]any)
	if !ok {
		return data
	}
	res, ok := rs["results"].(map[string]any)
	if !ok {
		return data
	}
	foods, ok := res["food"].([]any)
	if !ok {
		return data
	}
	for _, f := range foods {
		fm, ok := f.(map[string]any)
		if !ok {
			continue
		}
		sv, ok := fm["servings"].(map[string]any)
		if !ok {
			continue
		}
		trim := func(s map[string]any) map[string]any {
			out := map[string]any{}
			for _, k := range searchServingFields {
				if v, ok := s[k]; ok {
					out[k] = v
				}
			}
			return out
		}
		switch s := sv["serving"].(type) {
		case []any:
			for i, item := range s {
				if sm, ok := item.(map[string]any); ok {
					s[i] = trim(sm)
				}
			}
		case map[string]any:
			sv["serving"] = trim(s)
		}
	}
	return data
}
