package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gehan-malshan/matchmate/graphql-gateway/graph"
	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/config"
	requestcontext "github.com/gehan-malshan/matchmate/graphql-gateway/internal/transport"
	"github.com/gehan-malshan/matchmate/graphql-gateway/internal/upstream"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	client := upstream.New(cfg.Services)
	server := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{Upstream: client}}))
	server.AddTransport(transport.Options{})
	server.AddTransport(transport.GET{})
	server.AddTransport(transport.POST{})
	server.Use(extension.Introspection{})
	server.Use(extension.FixedComplexityLimit(250))
	server.SetErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
		presented := graphql.DefaultErrorPresenter(ctx, err)
		var problem *upstream.Problem
		if errors.As(err, &problem) {
			presented.Message = problem.Error()
			presented.Extensions = map[string]any{"code": problem.Code, "httpStatus": problem.Status}
		}
		return presented
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.Handle("/graphql", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		server.ServeHTTP(w, r.WithContext(requestcontext.WithHTTP(r.Context(), r, w)))
	}))
	if os.Getenv("APP_ENV") != "production" {
		mux.Handle("/", playground.Handler("MatchMate GraphQL", "/graphql"))
	}
	httpServer := &http.Server{Addr: cfg.Address, Handler: cors(cfg, mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	logger.Info("graphql_gateway_started", "address", cfg.Address)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("graphql_gateway_stopped", "error", err)
		os.Exit(1)
	}
}

func cors(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if cfg.AllowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Correlation-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
