# exoscale-metrics-collector

[![Build](https://img.shields.io/github/workflow/status/vshn/exoscale-metrics-collector/Test)][build]
![Go version](https://img.shields.io/github/go-mod/go-version/vshn/exoscale-metrics-collector)
[![Version](https://img.shields.io/github/v/release/vshn/exoscale-metrics-collector)][releases]
[![GitHub downloads](https://img.shields.io/github/downloads/vshn/exoscale-metrics-collector/total)][releases]

[build]: https://github.com/vshn/exoscale-metrics-collector/actions?query=workflow%3ATest
[releases]: https://github.com/vshn/exoscale-metrics-collector/releases

Batch job to sync usage data from the Exoscale API to the [APPUiO Cloud reporting](https://github.com/appuio/appuio-cloud-reporting/) database.

Metrics are collected taking into account product (e.g. `object-storage-storage:exoscale`), source (e.g. `exoscale:namespace`), tenant (as organization) and date time.

See the [component documentation](https://hub.syn.tools/exoscale-metrics-collector/index.html) for more information.

## Getting started for developers

In order to run this tool, you need
* An instance of the billing database
* Access to the Exoscale account which has the services to be invoiced
* Access to the Kubernetes cluster which has the claims corresponding to the Exoscale services

Get all this (see below), and put it all into an 'env' file:

```
export EXOSCALE_API_KEY="..."
export EXOSCALE_API_SECRET="..."
export K8S_SERVER_URL='https://...'
export K8S_TOKEN='...'
export ACR_DB_URL="postgres://reporting:reporting@localhost/appuio-cloud-reporting-test?sslmode=disable"
```

Then source the env file and run the client:

```
$ . ./env
$ make build
```

Then, run one of the available commands:

* Object Storage:
```
$ ./metrics-collector exoscale objectstorage
```

* DBaaS (runs metrics collector for all supported databases):
```
$ ./metrics-collector exoscale dbaas
```

### Billing Database

Provided that you have Docker installed, you can easily run a local instance of the billing database by getting the [appuio-cloud-reporting](https://github.com/appuio/appuio-cloud-reporting/) repository and running:

```
$ cd appuio-cloud-reporting
$ make docker-compose-up
```

For the first time you start the database locally, check `Local Development` on the Readme in [appuio-cloud-reporting](https://github.com/appuio/appuio-cloud-reporting/blob/master/README.md#local-development) or follow these steps:
* Next command asks for a password, it's "reporting":
```
$ createdb --username=reporting -h localhost -p 5432 appuio-cloud-reporting-test
```

* Then (no need to seed the database):
```
$ export ACR_DB_URL="postgres://reporting:reporting@localhost/appuio-cloud-reporting-test?sslmode=disable"
$ go run . migrate
```

### Create Resources in Lab Cluster to test metrics collector

You can first connect to your cluster and then create a claim for Postgres Database by applying a claim, for example:

```
apiVersion: appcat.vshn.io/v1
kind: ExoscalePostgreSQL
metadata:
  namespace: default
  name: exoscale-postgres-lab-test-1
spec:
  parameters:
    backup:
      timeOfDay: '13:00:00'
    maintenance:
      dayOfWeek: monday
      timeOfDay: "12:00:00"
    size:
      plan: hobbyist-2
    service:
      majorVersion: "14"
  writeConnectionSecretToRef:
    name: postgres-connection-details
```

Once the database is created and `Ready`, you can run locally the command:
```
$ ./metrics-collector exoscale dbaas
```

The same works for other resources. Just apply the right claim and run the proper command.

And don't forget to delete the resource(s) you created once you're done testing.

### Exoscale API key and secret

You can get your Exoscale API key and secret from the Exoscale web UI. Be sure to select the correct project.

The token should be restricted to the 'sos' and 'dbaas' services.

### Kubernetes API token

The following instructions work for OpenShift via the 'oc' utility. Not all of them will work with kubectl.

The commands assume that you are logged in to the Kubernetes cluster you want to use, and your working directory needs to be this git repository.

Instructions for OpenShift >=4.11:
```
$ cd exoscale-metrics-collector
$ oc -n default --as cluster-admin apply -f clusterrole.yaml 
$ oc -n default --as cluster-admin create serviceaccount vshn-exoscale-metrics-collector
$ oc --as cluster-admin adm policy add-cluster-role-to-user vshn-exoscale-metrics-collector system:serviceaccount:default:vshn-exoscale-metrics-collector
$ oc -n default --as cluster-admin apply -f clusterrole-secret.yaml
$ oc -n default --as cluster-admin get secret vshn-exoscale-metrics-collector-secret -o jsonpath='{.data.token}' | base64 -d
```

Instructions for OpenShift <=4.10:
```
$ cd exoscale-metrics-collector
$ oc -n default --as cluster-admin apply -f clusterrole.yaml 
$ oc -n default --as cluster-admin create serviceaccount vshn-exoscale-metrics-collector
$ oc --as cluster-admin adm policy add-cluster-role-to-user vshn-exoscale-metrics-collector system:serviceaccount:default:vshn-exoscale-metrics-collector
$ oc -n default --as cluster-admin serviceaccounts get-token vshn-exoscale-metrics-collector
```

The last command will print out your token without trailing newline; be sure to copy the correct part of the output.
