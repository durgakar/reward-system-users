package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/durgakar/reward-system-users/internal/app"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	application, err := app.New(*configPath)
	if err != nil {
		log.Fatalf("startup: %v", err)
	}

	fmt.Println("reward-system-users")
	fmt.Println(application.Describe())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	results, err := application.Runner.Run(ctx)
	if err != nil {
		log.Fatalf("campaign: %v", err)
	}

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
