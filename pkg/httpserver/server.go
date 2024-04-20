package httpserver

import (
	"context"
	"net"
	"net/http"
	"time"
)

func New(ctx context.Context, router http.Handler, port string) *http.Server {
	const defaultReadTimeout = time.Second * 30
	return &http.Server{
		Addr:    ":" + port,
		Handler: router,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		ReadTimeout: defaultReadTimeout,
	}
}
