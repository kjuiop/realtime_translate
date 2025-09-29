package app

import (
	"log"
	"realtime_translate/config"
	"realtime_translate/logger"
)

type App struct {
	cfg *config.Configuration
}

func NewApplication() *App {

	cfg, err := config.LoadConfiguration()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if err := logger.SlogInit(cfg.Logger); err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	return &App{
		cfg: cfg,
	}
}

func (a *App) Stop() {
}
