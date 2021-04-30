package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/IdeaEvolver/cutter-pkg/client"
	"github.com/IdeaEvolver/cutter-pkg/clog"
	"github.com/IdeaEvolver/cutter-pkg/service"
	"github.com/IdeaEvolver/cutter-status-dashboard/database"
	"github.com/IdeaEvolver/cutter-status-dashboard/healthchecks"
	"github.com/IdeaEvolver/cutter-status-dashboard/metrics"
	"github.com/IdeaEvolver/cutter-status-dashboard/server"
	"github.com/IdeaEvolver/cutter-status-dashboard/status"
	"github.com/kelseyhightower/envconfig"
	"go.opencensus.io/plugin/ochttp"
)

type Config struct {
	DbHost     string `envconfig:"DB_HOSTNAME" required:"true"`
	DbPort     string `envconfig:"DB_PORT" required:"true"`
	DbUsername string `envconfig:"DB_USERNAME" required:"true"`
	DbPassword string `envconfig:"DB_PASSWORD" required:"true"`
	DbName     string `envconfig:"DB_NAME" required:"true"`

	PlatformEndpoint       string `envconfig:"PLATFORM_ENDPOINT" required:"false"`
	FulfillmentHealthcheck string `envconfig:"FULFILLMENT_ENDPOINT" required:"false"`
	CrmHealthcheck         string `envconfig:"CRM_ENDPOINT" required:"false"`
	StudyHealthcheck       string `envconfig:"STUDY_ENDPOINT" required:"false"`

	GoogleProject string `envconfig:"GOOGLE_PROJECT" required:"true"`
	ClusterName   string `envconfig:"CLUSTER_NAME" required:"true"`

	PlatformDbHost     string `envconfig:"PLATFORM_DB_HOST" required:"true"`
	PlatformDbPort     string `envconfig:"PLATFORM_DB_PORT" required:"true"`
	PlatformDbUser     string `envconfig:"PLATFORM_DB_USERNAME" required:"true"`
	PlatformDbPassword string `envconfig:"PLATFORM_DB_PASSWORD" required:"true"`
	PlatformDbName     string `envconfig:"PLATFORM_DB_NAME" required:"true"`

	PORT string `envconfig:"PORT"`
}

func openDb(host, port, user, database, password string) *sql.DB {

	cs := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		host, port, user, database, password,
	)

	db, err := sql.Open("postgres", cs)
	if err != nil {
		clog.Fatalf("failed to connect to db")
	}

	clog.Infof("connected to postgres: %s:%s", host, port)

	return db
}

func main() {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		clog.Fatalf("config: %s", err)
	}

	db := openDb(
		cfg.DbHost,
		cfg.DbPort,
		cfg.DbUsername,
		cfg.DbName,
		cfg.DbPassword,
	)

	platformDb := openDb(
		cfg.PlatformDbHost,
		cfg.PlatformDbPort,
		cfg.PlatformDbUser,
		cfg.PlatformDbName,
		cfg.PlatformDbPassword,
	)

	databaseHealthchecker := database.NewHealthChecker(platformDb)

	statusStore := status.New(db)

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	internalClient := &http.Client{
		Transport: &ochttp.Transport{
			// Use Google Cloud propagation format.
			Propagation: &propagation.HTTPFormat{},
			Base:        customTransport,
		},
	}

	scfg := &service.Config{
		Addr:                fmt.Sprintf(":%s", cfg.PORT),
		ShutdownGracePeriod: time.Second * 10,
		MaxShutdownTime:     time.Second * 30,
	}

	healthchecksClient := &healthchecks.Client{
		Client:      client.New(internalClient),
		Platform:    cfg.PlatformEndpoint,
		Fulfillment: cfg.FulfillmentHealthcheck,
		Crm:         cfg.CrmHealthcheck,
		Study:       cfg.StudyHealthcheck,
	}

	metricsClient, err := metrics.New(cfg.GoogleProject, cfg.ClusterName)
	if err != nil {
		clog.Fatalf("unable to create metrics client: %v", err)
	}

	handler := &server.Handler{
		Healthchecks:   healthchecksClient,
		Statuses:       statusStore,
		Metrics:        metricsClient,
		DatabaseHealth: databaseHealthchecker,
	}
	s := server.New(scfg, handler)

	ctx := context.Background()
	go handler.AllChecks(ctx)

	clog.Infof("listening on %s", s.Addr)
	fmt.Println(s.ListenAndServe())
}
