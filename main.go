package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/josestg/env"
	"github.com/josestg/mux"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	type config struct {
		API struct {
			Host            string
			Port            int
			ReadTimeout     time.Duration
			WriteTimeout    time.Duration
			ShutdownTimeout time.Duration
		}
	}

	var cfg config

	c := env.NewConfig("Server", "WEBAPP")

	c.NewGroup("api",
		c.String(&cfg.API.Host, "host", "0.0.0.0"),
		c.Int(&cfg.API.Port, "port", 8080),
		c.Duration(&cfg.API.ReadTimeout, "read-timeout", 5*time.Second),
		c.Duration(&cfg.API.WriteTimeout, "write-timeout", 10*time.Second),
		c.Duration(&cfg.API.ShutdownTimeout, "shutdown-timeout", 15*time.Second),
	)

	if err := c.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	shutdownChannel := make(mux.ShutdownChannel, 1)
	requestIDMaker := new(RequestIDGenerator)

	router := mux.NewRouter(requestIDMaker, shutdownChannel)

	router.Get("/v1/readiness", handleReadiness)

	server := http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
		Handler:      router,
		ReadTimeout:  cfg.API.ReadTimeout,
		WriteTimeout: cfg.API.WriteTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Println("server is listening on:", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-shutdownChannel:
		ctx, cancel := context.WithTimeout(context.Background(), cfg.API.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Println("force shutdown")
			_ = server.Close()
		}
	case err := <-serverErr:
		if err != nil {
			log.Println("server is closed:", err.Error())
		}
	}
}

func handleReadiness(w http.ResponseWriter, r *http.Request) error {

	data := struct {
		Message string
	}{
		Message: "Server is ready.",
	}

	return mux.ToJSON(r.Context(), w, &data, http.StatusOK)
}

type RequestIDGenerator struct{}

func (RequestIDGenerator) NextRequestID() string {
	return uuid.NewString()
}
