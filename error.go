package krakend

import (
	"context"
	"errors"
	"net/http"
)

const (
	StatusClientDisconnected = 499
)

func ErrorToHTTPError(e error) int {
	if errors.Is(e, context.Canceled) {
		return StatusClientDisconnected
	}
	return http.StatusInternalServerError
}
