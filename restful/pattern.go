package restful

import "trpc.group/trpc-go/trpc-go/internal/httprule"

// Pattern makes *httprule.PathTemplate accessible.
type Pattern struct {
	*httprule.PathTemplate
}

// Parse parses the url path into a *Pattern. It should only be used by trpc-go-cmdline.
func Parse(urlPath string) (*Pattern, error) {
	tpl, err := httprule.Parse(urlPath)
	if err != nil {
		return nil, err
	}
	return &Pattern{tpl}, nil
}

// Enforce ensures the url path is legal (will panic if illegal) and parses it into a *Pattern.
func Enforce(urlPath string) *Pattern {
	pattern, err := Parse(urlPath)
	if err != nil {
		panic(err)
	}
	return pattern
}
