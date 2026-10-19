package provider

import (
	webclient "axual-webclient"
	"errors"
	"testing"
)

func TestIsGone(t *testing.T) {
	tests := map[string]struct {
		err  error
		gone bool
	}{
		"404":                    {webclient.NotFoundError, true},
		"application gone (400)": {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"No application found"}`}, true},
		"other 400":              {&webclient.HTTPError{StatusCode: 400, Body: `{"detail":"Invalid action"}`}, false},
		"403":                    {&webclient.HTTPError{StatusCode: 403, Body: `{"detail":"No application found"}`}, false},
		"500":                    {&webclient.HTTPError{StatusCode: 500}, false},
		"other error":            {errors.New("connection refused"), false},
		"no error":               {nil, false},
	}
	for name, test := range tests {
		if got := isGone(test.err); got != test.gone {
			t.Errorf("%s: isGone = %t, expected %t", name, got, test.gone)
		}
	}
}

func TestIgnoreGone(t *testing.T) {
	if err := ignoreGone(webclient.NotFoundError); err != nil {
		t.Errorf("ignoreGone(404) = %v, expected nil", err)
	}
	refused := &webclient.HTTPError{StatusCode: 403, Body: `{"detail":"Forbidden"}`}
	if err := ignoreGone(refused); err != refused {
		t.Errorf("ignoreGone(403) = %v, expected the 403", err)
	}
}
