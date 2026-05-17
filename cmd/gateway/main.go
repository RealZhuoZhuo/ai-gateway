package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"

	"github.com/RealZhuoZhuo/ai-gateway/internal/config"
	"github.com/RealZhuoZhuo/ai-gateway/internal/httpapi"
	"github.com/RealZhuoZhuo/ai-gateway/internal/providers"
	"github.com/RealZhuoZhuo/ai-gateway/internal/repo"
	"github.com/RealZhuoZhuo/ai-gateway/internal/service"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("load config failed")
	}
	if level, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
		logger.SetLevel(level)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pgRepo, err := repo.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		if errors.Is(err, repo.ErrNotConfigured) {
			logger.Warn("database_url not configured; only config api keys will be used")
		} else {
			logger.WithError(err).Fatal("postgres initialization failed")
		}
	}
	if pgRepo != nil {
		defer pgRepo.Close()
	}

	restyClient := resty.New().
		SetTimeout(cfg.HTTPTimeout).
		SetHeader("Accept", "application/json")
	ark := providers.NewArkClient(restyClient, cfg.ArkImageEndpoint, cfg.ArkImageAPIKey, cfg.ArkVideoEndpoint, cfg.ArkVideoAPIKey)
	kling := providers.NewKlingClient(restyClient, cfg.KlingBaseURL, cfg.KlingAccessKey, cfg.KlingSecretKey)
	dashscope := providers.NewDashScopeClient(restyClient, cfg.DashScopeBaseURL, cfg.DashScopeAPIKey)
	gateway := service.NewGateway(cfg, ark, kling, dashscope)
	handler := httpapi.NewHandler(gateway)
	authenticator := service.NewAuthenticator(cfg.GatewayAPIKeys, pgRepo)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewRouter(handler, authenticator, logger),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.WithField("addr", cfg.Addr).Info("gateway listening")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Error("server failed")
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Fatal("server shutdown failed")
	}
}
