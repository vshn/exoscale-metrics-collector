local kap = import 'lib/kapitan.libjsonnet';
local inv = kap.inventory();
local params = inv.parameters.metrics_collector;
local paramsACR = inv.parameters.appuio_cloud_reporting;
local kube = import 'lib/kube.libjsonnet';
local com = import 'lib/commodore.libjsonnet';
local collectorImage = '%(registry)s/%(repository)s:%(tag)s' % params.images.collector;
local alias = inv.parameters._instance;
local alias_suffix = '-' + alias;
local credentials_secret_name = 'credentials' + alias_suffix;
local component_name = 'metrics-collector';


local labels = {
  'app.kubernetes.io/name': component_name,
  'app.kubernetes.io/managed-by': 'commodore',
  'app.kubernetes.io/part-of': 'appuio-cloud-reporting',
  'app.kubernetes.io/component': component_name,
};

local secret(key) = [
  if params.secrets[key][s] != null then
    kube.Secret(s + alias_suffix) {
      metadata+: {
        namespace: paramsACR.namespace,
      },
    } + com.makeMergeable(params.secrets[key][s])
  for s in std.objectFields(params.secrets[key])
];

local cronjob(name, subcommand, schedule) = {
  kind: 'CronJob',
  apiVersion: 'batch/v1',
  metadata: {
    name: name,
    namespace: paramsACR.namespace,
    labels+: labels,
  },
  spec: {
    concurrencyPolicy: 'Forbid',
    failedJobsHistoryLimit: 5,
    jobTemplate: {
      spec: {
        template: {
          spec: {
            restartPolicy: 'OnFailure',
            containers: [
              {
                name: 'metrics-collector-backfill',
                image: collectorImage,
                args: [
                  subcommand,
                ],
                envFrom: [
                  {
                    secretRef: {
                      name: credentials_secret_name,
                    },
                  },
                ],
                env: [
                  {
                    name: 'password',
                    valueFrom: {
                      secretKeyRef: {
                        key: 'password',
                        name: 'reporting-db',
                      },
                    },
                  },
                  {
                    name: 'username',
                    valueFrom: {
                      secretKeyRef: {
                        key: 'username',
                        name: 'reporting-db',
                      },
                    },
                  },
                  {
                    name: 'ACR_DB_URL',
                    value: 'postgres://$(username):$(password)@%(host)s:%(port)s/%(name)s?%(parameters)s' % paramsACR.database,
                  },
                ],
                resources: {},
              },
            ],
          },
        },
      },
    },
    schedule: schedule,
    successfulJobsHistoryLimit: 3,
  },
};

assert params.exoscale.enabled != params.cloudscale.enabled : 'only one of the components can be enabled: cloudscale or exoscale. not both and not neither.';

(if params.exoscale.enabled then {
  local secrets = params.secrets['exoscale'],
  assert secrets != null : 'secrets must be set.',
  assert secrets.credentials != null : 'secrets.credentials must be set.',
  assert secrets.credentials.stringData != null : 'secrets.credentials.stringData must be set.',
  assert secrets.credentials.stringData.EXOSCALE_API_KEY != null : 'secrets.credentials.stringData.EXOSCALE_API_KEY must be set.',
  assert secrets.credentials.stringData.EXOSCALE_API_SECRET != null : 'secrets.credentials.stringData.EXOSCALE_API_SECRET must be set.',
  assert secrets.credentials.stringData.K8S_SERVER_URL != null : 'secrets.credentials.stringData.K8S_SERVER_URL must be set.',
  assert secrets.credentials.stringData.K8S_TOKEN != null : 'secrets.credentials.stringData.K8S_TOKEN must be set.',

  secrets: std.filter(function(it) it != null, secret('exoscale')),
  objectStorageCronjob: cronjob(alias + '-objectstorage', 'exoscale objectstorage', params.exoscale.objectStorage.schedule),
  [if params.exoscale.dbaas.enabled then 'dbaasCronjob']: cronjob(alias + '-dbaas', 'exoscale dbaas', params.exoscale.dbaas.schedule),
} else {})
+
(if params.cloudscale.enabled then {
  local secrets = params.secrets['cloudscale'],
  assert secrets != null : 'secrets must be set.',
  assert secrets.credentials != null : 'secrets.credentials must be set.',
  assert secrets.credentials.stringData != null : 'secrets.credentials.stringData must be set.',
  assert secrets.credentials.stringData.CLOUDSCALE_API_TOKEN != null : 'secrets.credentials.stringData.CLOUDSCALE_API_TOKEN must be set.',
  assert secrets.credentials.stringData.KUBERNETES_SERVER_URL != null : 'secrets.credentials.stringData.KUBERNETES_SERVER_URL must be set.',
  assert secrets.credentials.stringData.KUBERNETES_SERVER_TOKEN != null : 'secrets.credentials.stringData.KUBERNETES_SERVER_TOKEN must be set.',

  secrets: std.filter(function(it) it != null, secret('cloudscale')),
  [if params.cloudscale.objectStorage.enabled then 'objectStorageCronjob']: cronjob(alias + '-objectstorage', 'cloudscale objectstorage', params.cloudscale.objectStorage.schedule),
} else {})
