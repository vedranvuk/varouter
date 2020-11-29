// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Package servemux implements an API-compatible http.ServeMux alternative that
// uses Varouter internally. It serves mostly as an example of how to wrap
// Varouter into a custom mux.
package servemux

import (
	"context"
	"net/http"
	"sync"

	"github.com/vedranvuk/varouter"
)

// ServeMux is a serve mux that is API identical to http.ServeMux
// but is using varouter internally.
// The behaviour also mirrors Varouter's behaviour.
//
// Additionally, it stores any parsed Placeholders in a Placeholder map in the
// request context which is accessible via Placeholders helper function.
type ServeMux struct {
	mu sync.Mutex
	m  map[string]http.Handler
	r  *varouter.Varouter
}

// NewServeMux returns a new ServeMux instance.
func NewServeMux() *ServeMux {
	return &ServeMux{
		mu: sync.Mutex{},
		m:  make(map[string]http.Handler),
		r:  varouter.New(),
	}
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (mux *ServeMux) Handle(pattern string, handler http.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if err := mux.r.Register(pattern); err != nil {
		panic(err)
	}
	mux.m[pattern] = handler
}

// HandleFunc registers the handler function for the given pattern.
func (mux *ServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.Handle(pattern, http.HandlerFunc(handler))
}

// placeholderKey is a type of context key used to store Placeholder map.
type placeholderKey struct{ key string }

// placeholders is the Placeholders context key.
var placeholders = placeholderKey{"varouter/servemux/placeholders"}

// Placeholders is a helper method that retrieves Placeholders map from the
// context of a *http.Request. If no Placeholder map was stored in the request
// a nil Placeholders map is returned.
func Placeholders(r *http.Request) varouter.Vars {
	if placeholders, ok := r.Context().Value(placeholders).(varouter.Vars); ok {
		return placeholders
	}
	return nil
}

// Handler returns the handler to use for the given request,
// consulting r.URL.Path. It always returns a non-nil handler.
//
// If there is no registered handler that applies to the request,
// Handler returns a ``page not found'' handler and an empty pattern.
func (mux *ServeMux) Handler(r *http.Request) (h http.Handler, pattern string) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	templates, params, matched := mux.r.Match(r.URL.Path)
	if !matched {
		return http.NotFoundHandler(), ""
	}
	r = r.WithContext(context.WithValue(r.Context(), placeholders, params))
	pattern = templates[len(templates)-1]
	h = mux.m[pattern]
	return
}

// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, _ := mux.Handler(r)
	handler.ServeHTTP(w, r)
}
