// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type proxy struct {
	Name           string // to use in Via header
	RequestTimeout time.Duration
}

// headers that shouldn't be forwarded when proxying. See
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
var dropHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"TE":                true,
	"Trailer":           true,
	"Transfer-Encoding": true, // *-Encoding headers interfer with Go's automatic compression/decompression
	"Content-Encoding":  true,
	"Accept-Encoding":   true,
	"Upgrade":           true,
}

type ResponseFilter func(*http.Response, error) (*http.Response, error)

// Do sends a request, passes the result to the filter (if provided)
// and then if the result is not suppressed by the filter, sends the
// request to the ResponseWriter.  Returns true if a response was written,
// false if not.
func (p *proxy) Do(w http.ResponseWriter,
	reqIn *http.Request,
	urlOut *url.URL,
	client *http.Client,
	filter ResponseFilter) bool {

	// Copy headers from incoming request, then add/replace proxy
	// headers like Via and X-Forwarded-For.
	hdrOut := http.Header{}
	for k, v := range reqIn.Header {
		if !dropHeaders[k] {
			hdrOut[k] = v
		}
	}
	xff := reqIn.RemoteAddr
	if xffIn := reqIn.Header.Get("X-Forwarded-For"); xffIn != "" {
		xff = xffIn + "," + xff
	}
	hdrOut.Set("X-Forwarded-For", xff)
	if hdrOut.Get("X-Forwarded-Proto") == "" {
		hdrOut.Set("X-Forwarded-Proto", reqIn.URL.Scheme)
	}
	hdrOut.Add("Via", reqIn.Proto+" arvados-controller")

	ctx := reqIn.Context()
	if p.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(time.Duration(p.RequestTimeout)))
		defer cancel()
	}

	reqOut := (&http.Request{
		Method: reqIn.Method,
		URL:    urlOut,
		Host:   reqIn.Host,
		Header: hdrOut,
		Body:   reqIn.Body,
	}).WithContext(ctx)

	resp, err := client.Do(reqOut)
	if filter == nil && err != nil {
		httpserver.Error(w, err.Error(), http.StatusBadGateway)
		return true
	}

	// make sure original response body gets closed
	var originalBody io.ReadCloser
	if resp != nil {
		originalBody = resp.Body
		if originalBody != nil {
			defer originalBody.Close()
		}
	}

	if filter != nil {
		resp, err = filter(resp, err)

		if err != nil {
			httpserver.Error(w, err.Error(), http.StatusBadGateway)
			return true
		}
		if resp == nil {
			// filter() returned a nil response, this means suppress
			// writing a response, for the case where there might
			// be multiple response writers.
			return false
		}

		// the filter gave us a new response body, make sure that gets closed too.
		if resp.Body != originalBody {
			defer resp.Body.Close()
		}
	}

	for k, v := range resp.Header {
		for _, v := range v {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	n, err := io.Copy(w, resp.Body)
	if err != nil {
		httpserver.Logger(reqIn).WithError(err).WithField("bytesCopied", n).Error("error copying response body")
	}
	return true
}
