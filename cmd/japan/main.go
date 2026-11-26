package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"test/internal/web"
)

func main() {
	addr := flag.String("addr", ":8095", "address the web server listens on")
	flag.Parse()

	srv := web.NewServer()

	// Stop the server (and every live session) on Ctrl+C / SIGTERM.
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		log.Println("web: shutting down")
		srv.Shutdown()
	}()

	log.Printf("web app listening on %s, open http://localhost%s in your browser", *addr, *addr)
	if err := srv.ListenAndServe(*addr); err != nil {
		log.Fatalf("web: %v", err)
	}
}
