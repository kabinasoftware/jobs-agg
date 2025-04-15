package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	agg "github.com/kabinasoftware/jobs-agg"
	"github.com/kabinasoftware/jobs-agg/client/nofluffjobs"
	"github.com/kabinasoftware/jobs-agg/client/pracuj"
)

func main() {
	var (
		aggregator = agg.New(3)
		prw        = pracuj.Init(nil)
		nfw        = nofluffjobs.Init(nil)
	)

	aggregator.AddJob("pracuj-scraper", time.Hour, func(ctx context.Context) error {
		slog.Info("scraping pracuj.pl")

		pages, err := prw.GetPagesCount()
		if err != nil {
			slog.Error("failed to get pages count", "error", err)
			return err
		}
		if pages == 0 {
			slog.Info("pages count is equal to 0")
			return nil
		}

		for i := 1; i <= pages; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				offers, err := prw.GetOffers(i)
				if err != nil {
					slog.Error("failed to get offers", "error", err)
					return err
				}
				for _, offer := range offers {
					slog.Info("offer", "offer", offer)
				}
			}
		}

		return nil
	}, time.Now())

	aggregator.AddJob("nofluff-scraper", 30*time.Minute, func(ctx context.Context) error {
		slog.Info("scraping nofluffjobs")
		_ = nfw
		return nil
	}, time.Now())

	aggregator.AddJob("cleanup", 24*time.Hour, func(ctx context.Context) error {
		slog.Info("running cleanup")

		return nil
	}, time.Now())

	aggregator.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	aggregator.Stop()
}
