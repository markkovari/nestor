package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/markkovari/nestor/internal/config"
	"github.com/spf13/cobra"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run Nestor as a background server for webhooks and analysis",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	engine, database, err := initializeEngine(ctx, cfg)
	if err != nil {
		return err
	}
	if database != nil {
		defer database.Close(ctx)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "0.1.0"})
	})

	mux.HandleFunc("POST /analyze", func(w http.ResponseWriter, r *http.Request) {
		if err := engine.RunAnalysis(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /webhook/github", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "webhook read error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stdout, "webhook payload: %s\n", body)
		}
		go func() {
			bgCtx := context.Background()
			if err := engine.RunAnalysis(bgCtx); err != nil {
				fmt.Fprintf(os.Stderr, "webhook analysis error: %v\n", err)
			}
		}()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	})

	addr := fmt.Sprintf(":%d", servePort)
	fmt.Printf("Nestor server listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}
