package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"backend/internal/platform/logger"
)

// Server owns the HTTP listener and graceful shutdown.
type Server struct {
	http            *http.Server
	log             *logger.Logger
	shutdownTimeout time.Duration
}

func New(addr string, handler http.Handler, log *logger.Logger,
	readTimeout, writeTimeout, idleTimeout, shutdownTimeout time.Duration) *Server {
	return &Server{
		http: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		log:             log,
		shutdownTimeout: shutdownTimeout,
	}
}

// Run serves until ctx is cancelled, then shuts down gracefully. It returns
// nil for a clean shutdown.
func (s *Server) Run(ctx context.Context) error {
	serveErr := make(chan error, 1)
	go func() {
		s.log.Info("http server listening", "addr", s.http.Addr)
		serveErr <- s.http.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := s.http.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
