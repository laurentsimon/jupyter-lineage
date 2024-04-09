package http

import (
	"fmt"
	"net/url"
)

// TODO(#12): Sanitize the URL path
func AbsURLPath(p string) (string, error) {
	u, err := url.Parse(p)
	if err != nil {
		return "", fmt.Errorf("url parse: %w", err)
	}
	base, _ := url.Parse("")
	return base.ResolveReference(u).Path, nil
}
