// If the server is running as a Kubernetes deployment, remember to set the terminationGracePeriodSeconds accordingly

package main

import (
	"context"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

var failHealthChecks bool // you should use an atomic bool instead

func main() {

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	inflightTimeout, _ := context.WithTimeout(context.Background(), 15*time.Second)
	inflightCtx, gracefulStop := context.WithCancel(inflightTimeout)

	server := &http.Server{
		Addr:        "0.0.0.0:8080",
		BaseContext: func(l net.Listener) context.Context { return inflightCtx },
	}
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		if !failHealthChecks {
			w.Write([]byte("alive"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("server is shutting down"))
		}

	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.Write([]byte("OK"))
	})
  
	go func() {
		server.ListenAndServe()
	}()
  
	<-rootCtx.Done()
	stop()
	failHealthChecks = true
  // sleep for the polling period of liveness probes * threshold
	time.Sleep(10 * time.Second)
  
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	gracefulStop()
	if err != nil {
		time.Sleep(5 * time.Second)
	}
}
