package restservice

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/auth"
	"isc.org/stork/server/eventcenter"
	"isc.org/stork/server/metrics"
)

// Struct for holding response details.
type responseData struct {
	status int
	size   int
}

// Our http.ResponseWriter implementation.
type loggingResponseWriter struct {
	rw           http.ResponseWriter // compose original http.ResponseWriter
	responseData *responseData
}

// http.ResponseWriter Write implementation wrapper that captures size
// of the response.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// write response using original http.ResponseWriter
	size, err := r.rw.Write(b)
	// capture size
	r.responseData.size += size
	return size, err
}

// http.ResponseWriter WriteHeader implementation wrapper
// that captures status code of the response.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// write status code using original http.ResponseWriter
	r.rw.WriteHeader(statusCode)
	// capture status code
	r.responseData.status = statusCode
}

// http.ResponseWriter Header implementation wrapper
// that returns the header.
func (r *loggingResponseWriter) Header() http.Header {
	return r.rw.Header()
}

// Install a middleware that traces ReST calls using logrus.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteAddr := r.RemoteAddr
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			remoteAddr = realIP
		}
		entry := log.WithFields(log.Fields{
			"path":   r.RequestURI,
			"method": r.Method,
			"remote": remoteAddr,
		})

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lrw := &loggingResponseWriter{
			rw:           w, // compose original http.ResponseWriter
			responseData: responseData,
		}

		entry.Info("HTTP request incoming")

		start := time.Now()

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)

		entry = entry.WithFields(log.Fields{
			"status":      responseData.status,
			"text_status": http.StatusText(responseData.status),
			"took":        duration,
			"size":        responseData.size,
		})
		entry.Info("HTTP request served")
	})
}

// Install a middleware that is serving static files for UI
// and assets/pkgs content ie. stork rpm and deb packages.
func fileServerMiddleware(next http.Handler, staticFilesDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || r.URL.Path == "/swagger.json" {
			// serve API request
			next.ServeHTTP(w, r)
		} else {
			pth := path.Join(staticFilesDir, r.URL.Path)
			if _, err := os.Stat(pth); os.IsNotExist(err) {
				// if file does not exist then return content of index.html
				http.ServeFile(w, r, path.Join(staticFilesDir, "index.html"))
			} else {
				// if file exists then serve it
				http.FileServer(http.Dir(staticFilesDir)).ServeHTTP(w, r)
			}
		}
	})
}

// Install a middleware that is serving `server-sent events` (SSE).
func sseMiddleware(next http.Handler, eventCenter eventcenter.EventCenter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/sse") {
			eventCenter.ServeHTTP(w, r)
		} else {
			// pass request to another handler
			next.ServeHTTP(w, r)
		}
	})
}

// Install a middleware that is serving Agent installer.
func agentInstallerMiddleware(next http.Handler, staticFilesDir string) http.Handler {
	// Agent installer as Bash script.
	const agentInstallerScript = `#!/bin/bash
set -e -x

rm -f /tmp/isc-stork-agent.{deb,rpm,apk}

if [ -e /etc/debian_version ]; then
    curl -o /tmp/isc-stork-agent.deb "{{.ServerAddress}}{{.DebPath}}"
    DEBIAN_FRONTEND=noninteractive dpkg -i --force-confold /tmp/isc-stork-agent.deb
elif [ -e /etc/alpine-release ]; then
	wget -O /tmp/isc-stork-agent.apk "{{.ServerAddress}}{{.ApkPath}}"
	apk add --no-cache --no-network /tmp/isc-stork-agent.apk
else
    curl -o /tmp/isc-stork-agent.rpm "{{.ServerAddress}}{{.RpmPath}}"
    yum install -y /tmp/isc-stork-agent.rpm
fi

systemctl daemon-reload
systemctl enable isc-stork-agent
systemctl restart isc-stork-agent
systemctl status isc-stork-agent

su stork-agent -s /bin/sh -c 'stork-agent register -u http://{{.ServerAddress}}'

`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/stork-install-agent.sh") {
			pkgsDir := path.Join(staticFilesDir, "assets/pkgs")
			files, err := os.ReadDir(pkgsDir)
			if err != nil {
				msg := fmt.Sprintf("Problem reading '%s' directory with packages: %s\n", pkgsDir, err)
				log.Errorf(msg)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, msg)
				return
			}

			packageExtensions := []string{".deb", ".rpm", ".apk"}
			packageFiles := map[string]string{}
			for _, f := range files {
				if !strings.HasPrefix(f.Name(), "isc-stork-agent") {
					continue
				}

				for _, extension := range packageExtensions {
					if strings.HasSuffix(f.Name(), extension) {
						packageFiles[extension] = f.Name()
					}
				}
			}

			if len(packageFiles) != len(packageExtensions) {
				for _, extension := range packageExtensions {
					if _, ok := packageFiles[extension]; ok {
						continue
					}

					msg := fmt.Sprintf("Cannot find agent %s file in '%s' directory\n", extension, pkgsDir)
					log.Errorf(msg)
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprint(w, msg)
					return
				}
			}

			data := map[string]string{
				"ServerAddress": r.Host,
			}

			for extension, path := range packageFiles {
				key := strings.TrimLeft(extension, ".")
				key = strings.ToUpper(key[0:1]) + key[1:] + "Path"
				data[key] = path
			}

			t := template.Must(template.New("script").Parse(agentInstallerScript))
			err = t.Execute(w, data)
			if err != nil {
				msg := fmt.Sprintf("Problem preparing install script: %s\n", err)
				log.Errorf(msg)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, msg)
				return
			}
		} else {
			// pass request to another handler
			next.ServeHTTP(w, r)
		}
	})
}

// Metric collector middleware that handles the metric endpoint.
func metricsMiddleware(next http.Handler, collector metrics.Collector) http.Handler {
	var handler http.Handler
	if collector != nil {
		// Proper handler
		handler = collector.GetHTTPHandler(next)
	} else {
		// Placeholder handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			help := "The metrics collector endpoint is disabled."
			_, _ = w.Write([]byte(help))
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/metrics") {
			handler.ServeHTTP(w, r)
		} else {
			// pass request to another handler
			next.ServeHTTP(w, r)
		}
	})
}

// Global middleware function provides a common place to setup middlewares for
// the server. It is invoked before everything.
func (r *RestAPI) GlobalMiddleware(handler http.Handler, staticFilesDir string, eventCenter eventcenter.EventCenter) http.Handler {
	// last handler is executed first for incoming request
	handler = fileServerMiddleware(handler, staticFilesDir)
	handler = agentInstallerMiddleware(handler, staticFilesDir)
	handler = sseMiddleware(handler, eventCenter)
	handler = metricsMiddleware(handler, r.MetricsCollector)
	handler = loggingMiddleware(handler)
	return handler
}

// Inner middleware function provides a common place to setup middlewares for
// the server. It is invoked after routing but before authentication, binding and validation.
func (r *RestAPI) InnerMiddleware(handler http.Handler) http.Handler {
	// last handler is executed first for incoming request
	handler = r.SessionManager.SessionMiddleware(handler)
	return handler
}

// Checks if the user us authorized to access the system (has session).
func (r *RestAPI) Authorizer(req *http.Request) error {
	ok, u := r.SessionManager.Logged(req.Context())
	if !ok {
		return errors.Errorf("user unauthorized")
	}

	ok, _ = auth.Authorize(u, req)
	if !ok {
		return errors.Errorf("user logged in but not allowed to access the resource")
	}

	return nil
}
