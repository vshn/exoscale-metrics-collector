package main

import (
	"github.com/urfave/cli/v2"
	"github.com/vshn/exoscale-metrics-collector/pkg/clients/cluster"
	"github.com/vshn/exoscale-metrics-collector/pkg/clients/exoscale"
	"github.com/vshn/exoscale-metrics-collector/pkg/service/sos"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	objectStorageName = "objectstorage"
)

type objectStorageCommand struct {
	command
}

func newObjectStorageCommand() *cli.Command {
	command := &objectStorageCommand{}
	return &cli.Command{
		Name:   objectStorageName,
		Usage:  "Get metrics from object storage service",
		Action: command.execute,
		Flags: []cli.Flag{
			getClusterURLFlag(&command.clusterURL),
			getK8sServerTokenURLFlag(&command.clusterToken),
			getDatabaseURLFlag(&command.databaseURL),
			getExoscaleAccessKeyFlag(&command.exoscaleKey),
			getExoscaleSecretFlag(&command.exoscaleSecret),
		},
	}
}

func (c *objectStorageCommand) execute(ctx *cli.Context) error {
	log := AppLogger(ctx).WithName(objectStorageName)
	ctrl.SetLogger(log)

	log.Info("Creating Exoscale client")
	exoscaleClient, err := exoscale.InitClient(c.exoscaleKey, c.exoscaleSecret)
	if err != nil {
		return err
	}

	log.Info("Creating k8s client")
	k8sClient, err := cluster.InitK8sClient(c.clusterURL, c.clusterToken)
	if err != nil {
		return err
	}

	o := sos.NewObjectStorage(exoscaleClient, k8sClient, c.databaseURL)
	return o.Execute(ctx.Context)
}
