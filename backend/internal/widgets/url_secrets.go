package widgets

import (
	"net/url"
	"regexp"
)

// StripURLSecrets removes the query and the user info from every URL in a
// decoded widget config. Keys often live there (?apikey=..., private feed
// URLs), and neither an assistant nor a viewer of a shared board has use
// for them.
func StripURLSecrets(value any) any {
	return mapStrings(value, func(v string) string {
		if u, err := url.Parse(v); err == nil && u.Host != "" && (u.RawQuery != "" || u.User != nil) {
			u.RawQuery, u.User, u.Fragment = "", nil, ""
			return u.String()
		}
		return v
	})
}

// userInfo is the user info of a URL anywhere in a text (scheme://user:pw@),
// up to the last @ before the path, query or fragment
var userInfo = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^\s/?#]*@`)

// StripURLUserInfo removes only the user info from every URL in a decoded
// widget config, also from links in a text such as a note. The browser
// renders these configs, so the rest of a link stays.
func StripURLUserInfo(value any) any {
	return mapStrings(value, func(v string) string { return userInfo.ReplaceAllString(v, "${1}") })
}

// mapStrings replaces every string in a decoded JSON value with f's result
func mapStrings(value any, f func(string) string) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			v[key] = mapStrings(item, f)
		}
	case []any:
		for i, item := range v {
			v[i] = mapStrings(item, f)
		}
	case string:
		return f(v)
	}
	return value
}
