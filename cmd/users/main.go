// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains users main function to start the users service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	authclient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerclient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgclient "github.com/mainflux/mainflux/internal/clients/postgres"
	"github.com/mainflux/mainflux/internal/email"
	"github.com/mainflux/mainflux/internal/env"
	mfgroups "github.com/mainflux/mainflux/internal/groups"
	gapi "github.com/mainflux/mainflux/internal/groups/api"
	gevents "github.com/mainflux/mainflux/internal/groups/events"
	gpostgres "github.com/mainflux/mainflux/internal/groups/postgres"
	gtracing "github.com/mainflux/mainflux/internal/groups/tracing"
	"github.com/mainflux/mainflux/internal/postgres"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"
	capi "github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/emailer"
	uevents "github.com/mainflux/mainflux/users/events"
	"github.com/mainflux/mainflux/users/hasher"
	clientspg "github.com/mainflux/mainflux/users/postgres"
	ctracing "github.com/mainflux/mainflux/users/tracing"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "users"
	envPrefixDB    = "MF_USERS_DB_"
	envPrefixHTTP  = "MF_USERS_HTTP_"
	envPrefixGrpc  = "MF_USERS_GRPC_"
	defDB          = "users"
	defSvcHTTPPort = "9002"
	defSvcGRPCPort = "9192"
)

type config struct {
	LogLevel        string  `env:"MF_USERS_LOG_LEVEL"              envDefault:"info"`
	SecretKey       string  `env:"MF_USERS_SECRET_KEY"             envDefault:"secret"`
	AdminEmail      string  `env:"MF_USERS_ADMIN_EMAIL"            envDefault:""`
	AdminPassword   string  `env:"MF_USERS_ADMIN_PASSWORD"         envDefault:""`
	PassRegexText   string  `env:"MF_USERS_PASS_REGEX"             envDefault:"^.{8,}$"`
	AccessDuration  string  `env:"MF_USERS_ACCESS_TOKEN_DURATION"  envDefault:"15m"`
	RefreshDuration string  `env:"MF_USERS_REFRESH_TOKEN_DURATION" envDefault:"24h"`
	ResetURL        string  `env:"MF_TOKEN_RESET_ENDPOINT"         envDefault:"/reset-request"`
	JaegerURL       string  `env:"MF_JAEGER_URL"                   envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry   bool    `env:"MF_SEND_TELEMETRY"               envDefault:"true"`
	InstanceID      string  `env:"MF_USERS_INSTANCE_ID"            envDefault:""`
	ESURL           string  `env:"MF_USERS_ES_URL"                 envDefault:"redis://localhost:6379/0"`
	TraceRatio      float64 `env:"MF_JAEGER_TRACE_RATIO"           envDefault:"1.0"`
	PassRegex       *regexp.Regexp
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}
	passRegex, err := regexp.Compile(cfg.PassRegexText)
	if err != nil {
		log.Fatalf("invalid password validation rules %s\n", cfg.PassRegexText)
	}
	cfg.PassRegex = passRegex

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to init logger: %s", err.Error()))
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

	ec := email.Config{}
	if err := env.Parse(&ec); err != nil {
		logger.Error(fmt.Sprintf("failed to load email configuration : %s", err.Error()))
		exitCode = 1
		return
	}

	dbConfig := pgclient.Config{Name: defDB}
	if err := dbConfig.LoadEnv(envPrefixDB); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s database configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}

	cm := clientspg.Migration()
	gm := gpostgres.Migration()
	cm.Migrations = append(cm.Migrations, gm.Migrations...)
	db, err := pgclient.SetupWithConfig(envPrefixDB, *cm, dbConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	tp, err := jaegerclient.NewProvider(svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	auth, authHandler, err := authclient.Setup(svcName)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	csvc, gsvc, err := newService(ctx, auth, db, dbConfig, tracer, cfg, ec, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup service: %s", err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}

	mux := chi.NewRouter()
	httpSrv := httpserver.New(ctx, cancel, svcName, httpServerConfig, capi.MakeHandler(csvc, gsvc, mux, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return httpSrv.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSrv)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("users service terminated: %s", err))
	}
}

func newService(ctx context.Context, auth mainflux.AuthServiceClient, db *sqlx.DB, dbConfig pgclient.Config, tracer trace.Tracer, c config, ec email.Config, logger mflog.Logger) (users.Service, groups.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	cRepo := clientspg.NewRepository(database)
	gRepo := gpostgres.New(database)

	idp := uuid.New()
	hsr := hasher.New()

	emailer, err := emailer.New(c.ResetURL, &ec)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to configure e-mailing util: %s", err.Error()))
	}

	csvc := users.NewService(cRepo, auth, emailer, hsr, idp, c.PassRegex)
	gsvc := mfgroups.NewService(gRepo, idp, auth)

	csvc, err = uevents.NewEventStoreMiddleware(ctx, csvc, c.ESURL)
	if err != nil {
		return nil, nil, err
	}
	gsvc, err = gevents.NewEventStoreMiddleware(ctx, gsvc, c.ESURL)
	if err != nil {
		return nil, nil, err
	}

	csvc = ctracing.New(csvc, tracer)
	csvc = capi.LoggingMiddleware(csvc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	csvc = capi.MetricsMiddleware(csvc, counter, latency)

	gsvc = gtracing.New(gsvc, tracer)
	gsvc = gapi.LoggingMiddleware(gsvc, logger)
	counter, latency = internal.MakeMetrics("groups", "api")
	gsvc = gapi.MetricsMiddleware(gsvc, counter, latency)

	if err := createAdmin(ctx, c, cRepo, hsr, csvc); err != nil {
		logger.Error(fmt.Sprintf("failed to create admin client: %s", err))
	}

	return csvc, gsvc, err
}

func createAdmin(ctx context.Context, c config, crepo clientspg.Repository, hsr users.Hasher, svc users.Service) error {
	id, err := uuid.New().ID()
	if err != nil {
		return err
	}
	hash, err := hsr.Hash(c.AdminPassword)
	if err != nil {
		return err
	}

	client := mfclients.Client{
		ID:   id,
		Name: "admin",
		Credentials: mfclients.Credentials{
			Identity: c.AdminEmail,
			Secret:   hash,
		},
		Metadata: mfclients.Metadata{
			"role": "admin",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Role:      mfclients.AdminRole,
		Status:    mfclients.EnabledStatus,
	}

	if _, err := crepo.RetrieveByIdentity(ctx, client.Credentials.Identity); err == nil {
		return nil
	}

	// Create an admin
	if _, err = crepo.Save(ctx, client); err != nil {
		return err
	}
	if _, err = svc.IssueToken(ctx, c.AdminEmail, c.AdminPassword); err != nil {
		return err
	}

	return nil
}
