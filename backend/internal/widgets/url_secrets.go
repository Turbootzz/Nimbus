package widgets

import "net/url"

// StripURLSecrets removes the query and the user info from every URL in a
// decoded widget config. Keys often live there (?apikey=..., private feed
// URLs), and neither an assistant nor a viewer of a shared board has use
// for them.
func StripURLSecrets(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			v[key] = StripURLSecrets(item)
		}
	case []any:
		for i, item := range v {
			v[i] = StripURLSecrets(item)
		}
	case string:
		if u, err := url.Parse(v); err == nil && u.Host != "" && (u.RawQuery != "" || u.User != nil) {
			u.RawQuery, u.User, u.Fragment = "", nil, ""
			return u.String()
		}
	}
	return value
}
