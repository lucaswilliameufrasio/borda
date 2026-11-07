package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lucaswilliameufrasio/borda/internal/httpapi"
	"github.com/lucaswilliameufrasio/borda/internal/hub"
	"github.com/lucaswilliameufrasio/borda/internal/routeros"
	"github.com/lucaswilliameufrasio/borda/internal/secret"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8787", "loopback address")
	dataDir := flag.String("data", "data", "registry directory")
	readerName := flag.String("reader", "ssh", "ssh or memory")
	flag.Parse()

	if err := requireLoopback(*listen); err != nil {
		fmt.Fprintln(os.Stderr, "the listen address must be a loopback host")
		os.Exit(2)
	}
	reader, err := openReader(*readerName, *dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	keeper := secret.NewKeeper(*dataDir, secret.OSKeyring{}, nil)
	store, err := hub.Open(*dataDir, keeper, reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not open the router registry")
		os.Exit(1)
	}
	api := httpapi.New(store, log.New(os.Stderr, "", log.LstdFlags), []string{
		"http://127.0.0.1:5173",
		"http://localhost:5173",
	})
	server := &http.Server{
		Addr:              *listen,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	fmt.Fprintf(os.Stderr, "listening on http://%s\n", *listen)
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, "server stopped")
		os.Exit(1)
	}
}

func openReader(name string, dataDir string) (routeros.Reader, error) {
	if name == "memory" {
		reader := routeros.NewMemory()
		seedMemory(reader)
		return reader, nil
	}
	if name == "ssh" {
		return &routeros.SSHReader{KnownHostsPath: filepath.Join(dataDir, "known_hosts")}, nil
	}
	return nil, errors.New("reader must be ssh or memory")
}

func requireLoopback(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("not loopback")
	}
	return nil
}
