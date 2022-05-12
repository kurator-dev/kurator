package util

import (
	"net/url"
	"path"
)

// JoinUrlPath returns a URL string with the provided path elements joined to
// the existing path of base and the resulting path cleaned of any ./ or ../ elements.
func JoinUrlPath(base string, elem ...string) (result string, err error) {
	u, err := url.Parse(base)
	if err != nil {
		return
	}
	if len(elem) > 0 {
		elem = append([]string{u.Path}, elem...)
		u.Path = path.Join(elem...)
	}
	result = u.String()
	return
}
