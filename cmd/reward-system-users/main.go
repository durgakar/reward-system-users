package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/durgakar/reward-system-users/internal/app"
	"github.com/durgakar/reward-system-users/internal/campaign"
	"github.com/durgakar/reward-system-users/internal/config"
	"github.com/durgakar/reward-system-users/internal/server"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	configPath := fs.String("config", envOr("CONFIG_PATH", "config/config.yaml"), "path to config file")
	_ = fs.Parse(os.Args[2:])

	switch cmd {
	case "serve":
		runServe(*configPath)
	case "migrate":
		runMigrate(*configPath)
	case "run":
		runCampaign(*configPath)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("usage: reward-system-users <serve|migrate|run> [-config path]")
}

func runServe(configPath string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := cfg.ValidateProduction(true); err != nil {
		log.Fatalf("config: %v", err)
	}
	application, err := app.NewWithConfig(cfg)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	defer application.Close()

	srv := server.New(application, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartCron(ctx)

	httpServer := &http.Server{
		Addr:              application.Config.Addr(),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("listening on http://%s (admin UI: /admin)", application.Config.Addr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	srv.StopCron()
	_ = httpServer.Shutdown(shutdownCtx)
}

func runMigrate(configPath string) {
	application, err := app.New(configPath)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	defer application.Close()
	if err := application.Migrate("migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrations applied")
}

func runCampaign(configPath string) {
	application, err := app.New(configPath)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}
	defer application.Close()

	fmt.Println("reward-system-users")
	fmt.Println(application.Describe())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	results, err := application.RunCampaign(ctx)
	if err != nil {
		log.Fatalf("campaign: %v", err)
	}
	printResults(results)
}

func printResults(results []campaign.Result) {
	var totalPoints, totalEmails, totalErrors int
	for _, r := range results {
		totalPoints += r.PointsAwarded
		totalEmails += r.EmailsSent
		totalErrors += len(r.Errors)
		fmt.Printf("client=%s segments=%v rules=%d points=%d emails=%d errors=%d\n",
			r.ClientID, r.Segments, r.RuleMatches, r.PointsAwarded, r.EmailsSent, len(r.Errors))
		for _, e := range r.Errors {
			fmt.Fprintf(os.Stderr, "  error: %s\n", e)
		}
	}
	fmt.Printf("\nsummary: clients=%d points=%d emails=%d errors=%d\n",
		len(results), totalPoints, totalEmails, totalErrors)
	if totalErrors > 0 {
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
