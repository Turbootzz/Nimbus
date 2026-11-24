package utils

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithoutURL(t *testing.T) {
	cause := errors.New("connection refused")
	inner := &url.Error{Op: "parse", URL: "http://user:pass@host/?apikey=SECRET", Err: cause}
	outer := &url.Error{Op: "Get", URL: "http://user:pass@host/?apikey=SECRET", Err: inner}

	assert.Equal(t, cause, WithoutURL(outer))
	assert.Equal(t, cause, WithoutURL(fmt.Errorf("check: %w", outer)))
	assert.Equal(t, cause, WithoutURL(cause), "an error without a URL is left as it is")
}
