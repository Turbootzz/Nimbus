package utils

import (
	"errors"
	"net/url"
)

// WithoutURL returns the cause of a request error without the URL net/http
// puts in it, which can hold credentials (user:pass@, ?apikey=). The cause
// ("connection refused", a certificate error) is kept.
func WithoutURL(err error) error {
	var urlErr *url.Error
	for errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	return err
}
