package krakend

import (
	"context"
	"net/http"
)

const (
	StatusClientDisconnected = 499
)

func ErrorToHTTPError(e error) int {
	if e == context.Canceled {
		return StatusClientDisconnected
	}
	return http.StatusInternalServerError
}
