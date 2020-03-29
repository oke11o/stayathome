package main

import (
	"context"
	"go.uber.org/zap"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar().Named("stayathome_app").With("version", "v0.1.0")
	sugar.Info("The application is starting...")
	//plain := sugar.Desugar()

	port := os.Getenv("PORT")
	if port == "" {
		sugar.Fatal("PORT is not set")
	}

	diagPort := os.Getenv("DIAG_PORT")
	if diagPort == "" {
		sugar.Fatal("DIAG_PORT is not set")
	}

	r := mux.NewRouter()
	server := http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: r,
	}

	diagRouter := mux.NewRouter()
	diagRouter.HandleFunc("/health", func(
		w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	diag := http.Server{
		Addr:    net.JoinHostPort("", diagPort),
		Handler: diagRouter,
	}

	shutdown := make(chan error, 2)

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			shutdown <- err
		}
	}()

	go func() {
		err := diag.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			shutdown <- err
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case x := <-interrupt:
		sugar.Infow("Received", "signal", x.String())

	case err := <-shutdown:
		sugar.Errorw("Error from functional unit", "err", err.Error())
	}

	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	err := diag.Shutdown(timeout)
	if err != nil {
		// ?
	}

	err = server.Shutdown(timeout)
	if err != nil {
		// ?
	}

	sugar.Info("The application is shutdown")
}
