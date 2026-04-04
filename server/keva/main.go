package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zxdev/kvs/server/keva/router"
)

func main() {

	/*
		configure the router
	*/

	mux := http.NewServeMux()

	// on root request teapot status
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // 400
	})

	// heartbeat endpoint for 200 status
	mux.HandleFunc("/hb", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // 200
	})

	log.Println("server: keva format")
	var route router.KEVAServer
	mux.Handle("GET /create/{size}", route.CreateHandler())
	mux.Handle("GET /stats", route.StatusHandler())
	mux.Handle("GET /insert/{key}", route.InsertHandler())
	mux.Handle("POST /insert", route.InsertHandler())
	mux.Handle("GET /remove/{key}", route.RemoveHandler())
	mux.Handle("POST /remove", route.RemoveHandler())
	mux.Handle("GET /check/{key}", route.CheckHandler())
	mux.Handle("POST /check", route.CheckHandler())

	/*
		configure the http/https server
	*/

	var srv http.Server
	srv.ReadHeaderTimeout = time.Second * 5
	srv.ReadTimeout = time.Second * 10
	srv.WriteTimeout = time.Second * 30
	srv.IdleTimeout = time.Second * 15
	srv.Handler = mux

	srv.Addr = os.Getenv("HOST")
	if len(srv.Addr) == 0 {
		srv.Addr = "localhost:1455"
	}
	log.Println("server:", srv.Addr)
	go srv.ListenAndServe()

	/*
		graceful server shutdown
	*/

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	<-sig // wait
	srv.Shutdown(context.Background())
}
