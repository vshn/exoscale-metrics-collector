package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/vshn/exoscale-metrics-collector/pkg/clients/cluster"
	"github.com/vshn/exoscale-metrics-collector/pkg/clients/exoscale"
	"github.com/vshn/exoscale-metrics-collector/pkg/exoscale/dbaas"
	"github.com/vshn/exoscale-metrics-collector/pkg/exoscale/sos"
	"github.com/vshn/exoscale-metrics-collector/pkg/log"
	ctrl "sigs.k8s.io/controller-runtime"
)

func addCommandName(c *cli.Context) error {
	c.Context = log.NewLoggingContext(c.Context, log.AppLogger(c.Context).WithName(c.Command.Name))
	return nil
}

func ExoscaleCmds() *cli.Command {
	var (
		secret         string
		accessKey      string
		dbURL          string
		k8sServerToken string
		k8sServerURL   string
		kubeconfig     string
	)
	return &cli.Command{
		Name:  "exoscale",
		Usage: "Collect metrics from exoscale",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "exoscale-secret",
				Aliases:     []string{"s"},
				EnvVars:     []string{"EXOSCALE_API_SECRET"},
				Required:    true,
				Usage:       "The secret which has unrestricted SOS service access in an Exoscale organization",
				Destination: &secret,
			},
			&cli.StringFlag{
				Name:        "exoscale-access-key",
				Aliases:     []string{"k"},
				EnvVars:     []string{"EXOSCALE_API_KEY"},
				Required:    true,
				Usage:       "A key which has unrestricted SOS service access in an Exoscale organization",
				Destination: &accessKey,
			},
			&cli.StringFlag{
				Name:        "database-url",
				Aliases:     []string{"d"},
				EnvVars:     []string{"ACR_DB_URL"},
				Required:    true,
				Usage:       "A PostgreSQL database URL where to save relevant metrics",
				Destination: &dbURL,
			},
			&cli.StringFlag{
				Name:        "k8s-server-token",
				Aliases:     []string{"t"},
				EnvVars:     []string{"K8S_TOKEN"},
				Required:    true,
				Usage:       "A Kubernetes server token which can view buckets.exoscale.crossplane.io resources",
				Destination: &k8sServerToken,
			},
			&cli.StringFlag{
				Name:        "k8s-server-url",
				Aliases:     []string{"u"},
				EnvVars:     []string{"K8S_SERVER_URL"},
				Required:    true,
				Usage:       "A Kubernetes server URL from where to get the data from",
				Destination: &k8sServerURL,
			},
			&cli.StringFlag{
				Name:        "kubeconfig",
				EnvVars:     []string{"KUBECONFIG"},
				Usage:       "Path to a kubeconfig file which will be used instead of url/token flags if set",
				Destination: &kubeconfig,
			},
		},
		Before: addCommandName,
		Subcommands: []*cli.Command{
			{
				Name:   "objectstorage",
				Usage:  "Get metrics from object storage service",
				Before: addCommandName,
				Action: func(c *cli.Context) error {
					logger := log.AppLogger(c.Context)
					ctrl.SetLogger(logger)

					logger.Info("Creating Exoscale client")
					exoscaleClient, err := exoscale.InitClient(accessKey, secret)
					if err != nil {
						return fmt.Errorf("exoscale client: %w", err)
					}

					logger.Info("Creating k8s client")
					k8sClient, err := cluster.NewClient(kubeconfig, k8sServerURL, k8sServerToken)
					if err != nil {
						return fmt.Errorf("k8s client: %w", err)
					}

					o, err := sos.NewObjectStorage(exoscaleClient, k8sClient, dbURL)
					if err != nil {
						return fmt.Errorf("object storage: %w", err)
					}
					return o.Execute(c.Context)
				},
			},
			{
				Name:   "dbaas",
				Usage:  "Get metrics from database service",
				Before: addCommandName,
				Action: func(c *cli.Context) error {
					logger := log.AppLogger(c.Context)
					ctrl.SetLogger(logger)

					logger.Info("Creating Exoscale client")
					exoscaleClient, err := exoscale.InitClient(accessKey, secret)
					if err != nil {
						return fmt.Errorf("exoscale client: %w", err)
					}

					logger.Info("Creating k8s client")
					k8sClient, err := cluster.NewDynamicClient(kubeconfig, k8sServerURL, k8sServerToken)
					if err != nil {
						return fmt.Errorf("k8s client: %w", err)
					}

					o, err := dbaas.NewDBaaSService(exoscaleClient, k8sClient, dbURL)
					if err != nil {
						return fmt.Errorf("dbaas service: %w", err)
					}
					return o.Execute(c.Context)
				},
			},
		},
	}
}
