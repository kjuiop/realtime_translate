package main

import (
	"log/slog"
	"os"
	"os/signal"
	"realtime_translate/cmd/app"
	"syscall"
)

var BUILD_TIME = "no flag of BUILD_TIME"
var GIT_HASH = "no flag of GIT_HASH"
var APP_VERSION = "no flag of APP_VERSION"

func main() {

	a := app.NewApplication()

	slog.Debug("realtime translate app start", "git_hash", GIT_HASH, "build_time", BUILD_TIME, "app_version", APP_VERSION)

	<-exitSignal()
	a.Stop()
	slog.Debug("realtime translate app gracefully stopped")
}

func exitSignal() <-chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	return sig
}
