// Command flowcastd is a web service: users upload a table and get back a
// .drawio file.
//
//	flowcastd [-addr :8080] [-concurrent 4]
//
// POST /api/build accepts multipart/form-data with a file field, plus the optional
// fields title, sheet, delimiter and encoding. It returns JSON with issues,
// warnings and the .drawio content; with ?download=1 it returns the file directly
// when the build succeeds. POST /api/check only checks the table. GET / is the
// upload page.
//
// The service uses the WebLimits limit profile, does not merge and does not call
// drawio: it neither reads nor writes files and runs no external processes.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/luytbq/flowcast"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	concurrent := flag.Int("concurrent", runtime.NumCPU(), "number of builds running concurrently")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	srv := &http.Server{
		Addr:              *addr,
		Handler:           newServer(flowcast.WebLimits, *concurrent, log),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		srv.Shutdown(shut)
	}()
	log.Info("flowcastd listening", "addr", *addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("stopped", "err", err)
		os.Exit(1)
	}
}
