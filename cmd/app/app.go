package app

import (
	"context"
	"log"
	"realtime_translate/config"
	"realtime_translate/logger"
	"realtime_translate/server"
)

type App struct {
	cfg  *config.Configuration
	rtmp *server.RTMPServer
}

func NewApplication(ctx context.Context) *App {

	cfg, err := config.LoadConfiguration()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := logger.SlogInit(cfg.Logger); err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	rtmp, err := server.NewRTMPServer(ctx, cfg.RTMPServer)
	if err != nil {
		log.Fatalf("failed to init rtmp server: %v", err)
	}

	return &App{
		cfg:  cfg,
		rtmp: rtmp,
	}
}

func (a *App) ListenRtmp(ctx context.Context) {
	a.rtmp.ListenRtmp()
}

func (a *App) Stop() {
}
