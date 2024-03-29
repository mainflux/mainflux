// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains websocket-adapter main function to start the websocket-adapter service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	authapi "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerclient "github.com/mainflux/mainflux/internal/clients/jaeger"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	brokerstracing "github.com/mainflux/mainflux/pkg/messaging/brokers/tracing"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/ws"
	"github.com/mainflux/mainflux/ws/api"
	"github.com/mainflux/mainflux/ws/tracing"
	"github.com/mainflux/mproxy/pkg/session"
	"github.com/mainflux/mproxy/pkg/websockets"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "ws-adapter"
	envPrefixHTTP  = "MF_WS_ADAPTER_HTTP_"
	defSvcHTTPPort = "8190"
	targetWSPort   = "8191"
	targetWSHost   = "localhost"
)

type config struct {
	LogLevel      string  `env:"MF_WS_ADAPTER_LOG_LEVEL"    envDefault:"info"`
	BrokerURL     string  `env:"MF_MESSAGE_BROKER_URL"      envDefault:"nats://localhost:4222"`
	JaegerURL     string  `env:"MF_JAEGER_URL"              envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry bool    `env:"MF_SEND_TELEMETRY"          envDefault:"true"`
	InstanceID    string  `env:"MF_WS_ADAPTER_INSTANCE_ID"  envDefault:""`
	TraceRatio    float64 `env:"MF_JAEGER_TRACE_RATIO"      envDefault:"1.0"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	var exitCode int
	defer mflog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	targetServerConf := server.Config{
		Port: targetWSPort,
		Host: targetWSHost,
	}

	auth, aHandler, err := authapi.SetupAuthz("authz")
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer aHandler.Close()

	logger.Info("Successfully connected to things grpc server " + aHandler.Secure())

	tp, err := jaegerclient.NewProvider(svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	nps, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer nps.Close()
	nps = brokerstracing.NewPubSub(targetServerConf, tracer, nps)

	svc := newService(auth, nps, logger, tracer)

	hs := httpserver.New(ctx, cancel, svcName, targetServerConf, api.MakeHandler(ctx, svc, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		g.Go(func() error {
			return hs.Start()
		})
		handler := ws.NewHandler(nps, logger, auth)
		return proxyWS(ctx, httpServerConfig, logger, handler)
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("WS adapter service terminated: %s", err))
	}
}

func newService(tc mainflux.AuthzServiceClient, nps messaging.PubSub, logger mflog.Logger, tracer trace.Tracer) ws.Service {
	svc := ws.New(tc, nps)
	svc = tracing.New(tracer, svc)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("ws_adapter", "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	return svc
}

func proxyWS(ctx context.Context, cfg server.Config, logger mflog.Logger, handler session.Handler) error {
	target := fmt.Sprintf("ws://%s:%s", targetWSHost, targetWSPort)
	address := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	wp, err := websockets.NewProxy(address, target, logger, handler)
	if err != nil {
		return err
	}

	errCh := make(chan error)

	go func() {
		if cfg.CertFile != "" && cfg.KeyFile != "" {
			logger.Info(fmt.Sprintf("ws-adapter service http server listening at %s:%s with TLS", cfg.Host, cfg.Port))
			errCh <- wp.ListenTLS(cfg.CertFile, cfg.KeyFile)
		} else {
			logger.Info(fmt.Sprintf("ws-adapter service http server listening at %s:%s without TLS", cfg.Host, cfg.Port))
			errCh <- wp.Listen()
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT WS shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}
}
