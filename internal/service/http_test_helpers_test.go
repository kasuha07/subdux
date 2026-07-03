package service

import "net/http"

type notificationTestRoundTripper func(req *http.Request) (*http.Response, error)

func (fn notificationTestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
