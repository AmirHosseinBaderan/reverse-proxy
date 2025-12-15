// Package host provides HTTP request routing based on host headers.
// It implements host-based virtual hosting for the reverse proxy.
package host

import (
	"net/http"
	"strings"
)

func Router(sites map[string]http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.Split(r.Host, ":")[0] // remove port

		if h, ok := sites[host]; ok {
			h.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})
}
