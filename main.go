package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/IdeaEvolver/cutter-pkg/client"
	"github.com/IdeaEvolver/cutter-pkg/clog"
	"github.com/IdeaEvolver/cutter-pkg/service"
	"github.com/IdeaEvolver/cutter-status-dashboard/healthchecks"
	"github.com/IdeaEvolver/cutter-status-dashboard/metrics"
	"github.com/IdeaEvolver/cutter-status-dashboard/server"
	"github.com/IdeaEvolver/cutter-status-dashboard/status"
	"github.com/kelseyhightower/envconfig"
	"go.opencensus.io/plugin/ochttp"

	"contrib.go.opencensus.io/integrations/ocsql"
)

type Config struct {
	DbHost     string `envconfig:"DB_HOSTNAME" required:"true"`
	DbPort     string `envconfig:"DB_PORT" required:"true"`
	DbUsername string `envconfig:"DB_USERNAME" required:"true"`
	DbPassword string `envconfig:"DB_PASSWORD" required:"true"`
	DbName     string `envconfig:"DB_NAME" required:"true"`
	DbOpts     string `envconfig:"DB_OPTS" required:"false"`

	PlatformEndpoint       string `envconfig:"PLATFORM_ENDPOINT" required:"false"`
	FulfillmentHealthcheck string `envconfig:"FULFILLMENT_ENDPOINT" required:"false"`
	CrmHealthcheck         string `envconfig:"CRM_ENDPOINT" required:"false"`
	StudyHealthcheck       string `envconfig:"STUDY_ENDPOINT" required:"false"`

	GoogleProject string `envconfig:"GOOGLE_PROJECT" required:"true"`
	ClusterName   string `envconfig:"CLUSTER_NAME" required:"true"`
	BucketName    string `envconfig:"BUCKET_NAME" required:"true"`

	PORT string `envconfig:"PORT"`
}

func main() {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		clog.Fatalf("config: %s", err)
	}

	cs := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s password=%s sslmode=disable",
		cfg.DbHost,
		cfg.DbPort,
		cfg.DbUsername,
		cfg.DbName,
		cfg.DbPassword,
	)

	driverName, err := ocsql.Register(
		"postgres",
		ocsql.WithQuery(true),
		ocsql.WithQueryParams(true),
		ocsql.WithInstanceName("status-dashboard"),
	)
	if err != nil {
		clog.Fatalf("unable to register our ocsql driver: %v\n", err)
	}

	db, err := sql.Open(driverName, cs)
	if err != nil {
		clog.Fatalf("failed to connect to db")
	}

	clog.Infof("connected to postgres: %s:%s", cfg.DbHost, cfg.DbPort)

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

	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		clog.Fatalf("unable to create storage client: %v", err)
	}

	handler := &server.Handler{
		Healthchecks: healthchecksClient,
		Statuses:     statusStore,
		Metrics:      metricsClient,
		Storage:      storageClient,
	}
	s := server.New(scfg, handler)

	go handler.AllChecks(ctx, cfg.BucketName)

	clog.Infof("listening on %s", s.Addr)
	fmt.Println(s.ListenAndServe())
}
