package host

import (
	"net/http"
	"reverse-proxy/internal/models/global"
	"strings"
)

func HostRouter(sites map[string]*global.SiteConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.Split(r.Host, ":")[0] // remove port

		if h, ok := sites[host]; ok {

		}

		http.NotFound(w, r)
	})
}
