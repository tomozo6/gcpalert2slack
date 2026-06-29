package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tomozo6/gcpalert2slack/internal/config"
	"github.com/tomozo6/gcpalert2slack/internal/server"
	"github.com/tomozo6/gcpalert2slack/internal/slacknotify"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf(".env was not loaded: %v", err)
	}

	cfg := config.Load()
	if cfg.SlackBotToken == "" || cfg.SlackChannelID == "" {
		log.Fatalf("load config: %v", fmt.Errorf("SLACK_BOT_TOKEN and SLACK_CHANNEL_ID are required"))
	}

	handler := server.NewHandler(slacknotify.NewClient(cfg.SlackBotToken, cfg.SlackChannelID))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown server: %v", err)
		}
	}()

	log.Printf("listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
