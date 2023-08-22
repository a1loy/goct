package healthcheck

import (
	"context"
	"fmt"
	"goct/internal/logger"
	"net/http"
	"time"
)

const (
	Host     = "localhost"
	Port     = "8081"
	Endpoint = "ping"
)

func handler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}

func RunHealthCheck(ctx context.Context, host string, port string, endpoint string) {
	logger.Infof("running healthcheck on %s:%s/%s", host, port, endpoint)
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/%s", endpoint), handler)
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", host, port),
		Handler: mux,
	}
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	srv.ListenAndServe()
}
