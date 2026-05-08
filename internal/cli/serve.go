package cli

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/markkovari/nestor/internal/config"
	"github.com/spf13/cobra"
)

var servePort int
var webhookSecret string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run Nestor as a background server for webhooks and analysis",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	serveCmd.Flags().StringVar(&webhookSecret, "webhook-secret", "", "GitHub webhook secret for signature verification")
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
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "0.1.0"})
	})

	mux.HandleFunc("POST /analyze", func(w http.ResponseWriter, r *http.Request) {
		if err := engine.RunAnalysis(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /webhook/github", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sig := r.Header.Get("X-Hub-Signature-256")
		if !verifyGitHubSignature(webhookSecret, body, sig) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid signature"})
			return
		}
		go func() {
			bgCtx := context.Background()
			if err := engine.RunAnalysis(bgCtx); err != nil {
				fmt.Fprintf(os.Stderr, "webhook analysis error: %v\n", err)
			}
		}()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	})

	addr := fmt.Sprintf(":%d", servePort)
	fmt.Printf("Nestor server listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func verifyGitHubSignature(secret string, body []byte, sigHeader string) bool {
	if secret == "" {
		return true
	}
	if !strings.HasPrefix(sigHeader, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sigHeader))
}
