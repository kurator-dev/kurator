local t = import 'kube-thanos/thanos.libsonnet';

local commonConfig = {
  config+:: {
    local cfg = self,
    namespace: 'thanos',
    version: 'v0.24.0',
    image: 'quay.io/thanos/thanos:' + cfg.version,
    imagePullPolicy: 'IfNotPresent',
    objectStorageConfig: {
      name: 'thanos-objstore-config',
      key: 'thanos.yaml',
    },
    hashringConfigMapName: 'hashring-config',
    volumeClaimTemplate: {
      spec: {
        accessModes: ['ReadWriteOnce'],
        resources: {
          requests: {
            storage: '10Gi',
          },
        },
      },
    },
  },
};

local i = t.receiveIngestor(commonConfig.config {
  replicas: 1,
  replicaLabels: ['receive_replica'],
  replicationFactor: 1,
  // Disable shipping to object storage for the purposes of this example
  objectStorageConfig: null,
});

local r = t.receiveRouter(commonConfig.config {
  replicas: 1,
  replicaLabels: ['receive_replica'],
  replicationFactor: 1,
  // Disable shipping to object storage for the purposes of this example
  objectStorageConfig: null,
  endpoints: i.endpoints,
});

local s = t.store(commonConfig.config {
  replicas: 1,
  serviceMonitor: true,
});

local sc = t.sidecar(commonConfig.config {
  name: 'prometheus-thanos-thanos-sidecar',
  namespace: 'monitoring',
  serviceMonitor: true,
  // Labels of the Prometheus pods with a Thanos Sidecar container
  podLabelSelector: {
    // Here it is the default label given by the prometheus-operator
    // to all Prometheus pods
    'app.kubernetes.io/name': 'prometheus',
  },
});

local q = t.query(commonConfig.config {
  name: 'thanos-query',
  replicas: 1,
  externalPrefix: '',
  resources: {},
  queryTimeout: '5m',
  autoDownsampling: true,
  lookbackDelta: '15m',
  replicaLabels: ['prometheus_replica', 'rule_replica'],
  ports: {
    grpc: 10901,
    http: 9090,
  },
  serviceMonitor: true,
  logLevel: 'debug',
});

// add for thanos-sidecar-remote service
local sr = t.sidecar(commonConfig.config {
  name: 'thanos-sidecar-remote',
  namespace: 'thanos',
});

local finalQ = t.query(q.config {
  stores: [
    'dnssrv+_grpc._tcp.%s.%s.svc.cluster.local' % [service.metadata.name, service.metadata.namespace]
    for service in [sc.service, s.service, sr.service]
  ] + i.storeEndpoints,
});

{ ['thanos-store-' + name]: s[name] for name in std.objectFields(s) } +
{ ['thanos-query-' + name]: finalQ[name] for name in std.objectFields(finalQ) if finalQ[name] != null } +
{ ['thanos-receive-router-' + resource]: r[resource] for resource in std.objectFields(r) } +
{ ['thanos-receive-ingestor-' + resource]: i[resource] for resource in std.objectFields(i) if resource != 'ingestors' } +
{
  ['thanos-receive-ingestor-' + hashring + '-' + resource]: i.ingestors[hashring][resource]
  for hashring in std.objectFields(i.ingestors)
  for resource in std.objectFields(i.ingestors[hashring])
  if i.ingestors[hashring][resource] != null
}