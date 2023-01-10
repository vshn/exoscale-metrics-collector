package cmd

import (
	"fmt"
	"net/http"

	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/urfave/cli/v2"
	"github.com/vshn/exoscale-metrics-collector/pkg/clients/cluster"
	cs "github.com/vshn/exoscale-metrics-collector/pkg/cloudscale"
	"github.com/vshn/exoscale-metrics-collector/pkg/log"
	ctrl "sigs.k8s.io/controller-runtime"
)

func CloudscaleCmds() *cli.Command {
	var (
		apiToken              string
		dbURL                 string
		kubernetesServerToken string
		kubernetesServerURL   string
		kubeconfig            string
		days                  int
	)
	return &cli.Command{
		Name:  "cloudscale",
		Usage: "Collect metrics from cloudscale",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "cloudscale-api-token",
				EnvVars:     []string{"CLOUDSCALE_API_TOKEN"},
				Required:    true,
				Usage:       "API token for cloudscale",
				Destination: &apiToken,
			},
			&cli.StringFlag{
				Name:        "database-url",
				EnvVars:     []string{"ACR_DB_URL"},
				Required:    true,
				Usage:       "A PostgreSQL database URL where to save relevant metrics",
				Destination: &dbURL,
			},
			&cli.StringFlag{
				Name:        "kubernetes-server-url",
				EnvVars:     []string{"KUBERNETES_SERVER_URL"},
				Required:    true,
				Usage:       "A Kubernetes server URL from where to get the data from",
				Destination: &kubernetesServerURL,
			},
			&cli.StringFlag{
				Name:        "kubernetes-server-token",
				EnvVars:     []string{"KUBERNETES_SERVER_TOKEN"},
				Required:    true,
				Usage:       "A Kubernetes server token which can view buckets.cloudscale.crossplane.io resources",
				Destination: &kubernetesServerToken,
			},
			&cli.StringFlag{
				Name:        "kubeconfig",
				EnvVars:     []string{"KUBECONFIG"},
				Usage:       "Path to a kubeconfig file which will be used instead of url/token flags if set",
				Destination: &kubeconfig,
			},
			&cli.IntFlag{
				Name:        "days",
				EnvVars:     []string{"DAYS"},
				Required:    true,
				Usage:       "Days of metrics to fetch since today",
				Destination: &days,
			},
		},
		Before: addCommandName,
		Subcommands: []*cli.Command{
			{
				Name:   "objectstorage",
				Usage:  "Get metrics from object storage service",
				Before: addCommandName,
				Action: func(c *cli.Context) error {
					log := log.AppLogger(c.Context)
					ctrl.SetLogger(log)

					log.Info("Creating cloudscale client")
					cloudscaleClient := cloudscale.NewClient(http.DefaultClient)
					cloudscaleClient.AuthToken = apiToken

					log.Info("Creating k8s client")
					k8sClient, err := cluster.NewClient(kubeconfig, kubernetesServerURL, kubernetesServerToken)
					if err != nil {
						return fmt.Errorf("k8s client: %w", err)
					}

					o := cs.NewObjectStorage(cloudscaleClient, k8sClient, days, dbURL)
					return o.Execute(c.Context)
				},
			},
		},
	}
}
