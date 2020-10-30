+++
title = "Getting Started"
description = "The easiest way to start with kube profefe"
weight = 12
draft = false
toc = true
bref = ""
+++
## Goal

The goal for this tutorial is to gather our first profile from a running pod via
kubectl profefe plugin. We will store the profile locally, and we will push one
to profefe

## Prerequisites

If you have those things already done move to the chapter "getting started"

1. Have a Kubernetes cluster up and running
2. Deploy profefe inside or outside your Kubernetes cluster. But it has to be
   reachable from the Kubernetes cluster network. I will assume that the URL for
   it is `https://profefe.internal.company.com:10100`
3. You should have the kubectl profefe plugin already installed (checkout the
   installation doc)

### Deploy kubernetes cluster with KinD

```
$ kind create cluster
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.15.3) 🖼
 ✓ Preparing nodes 📦
 ✓ Creating kubeadm config 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
Cluster creation complete. You can now use the cluster with:

export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"
kubectl cluster-info

$ export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"
✔ ~/git/kube-profefe [docs/getting-start L|✚ 1…1]

$ kubectl cluster-info
Kubernetes master is running at https://127.0.0.1:42659
KubeDNS is running at https://127.0.0.1:42659/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```

### Deploy profefe

Apply this spec via `kubectl apply -f <path file.yaml>`. It will create a
profefe namespace, and it will deploy the
[profefe](https://github.com/profefe/profefe) as a deployment with its service.

This installation is not for production.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: profefe
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    component: profefe
  name: profefe
  namespace: profefe
spec:
  replicas: 1
  selector:
    matchLabels:
      component: profefe
  template:
    metadata:
      labels:
        component: profefe
    spec:
      containers:
      - args:
        - -badger.dir
        - /tmp/profefe-data
        image: profefe/profefe:git-668a19d
        imagePullPolicy: IfNotPresent
        name: profefe
        ports:
        - containerPort: 10100
---
apiVersion: v1
kind: Service
metadata:
  labels:
    component: profefe
  name: profefe-collector
  namespace: profefe
spec:
  ports:
  - name: collector
    port: 10100
    protocol: TCP
    targetPort: 10100
  selector:
    component: profefe
  type: ClusterIP
```

### Getting Started

Do you have the kubectl plugin? Check it out with:

```
$ kubectl profefe --help
It is a kubectl plugin that you can use to retrieve and manage profiles in Go.

Usage:
  kubectl-profefe [flags]
  kubectl-profefe [command]

Available Commands:
  capture     Capture gathers profiles for a pod or a set of them. If can filter by namespace and via label selector.
  get         Display one or many resources
  help        Help about any command
  load        Load a profile you have locally to profefe

Flags:
  -A, --all-namespaces                 If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default HTTP cache directory (default "/home/gianarb/.kube/http-cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
  -f, --filename strings               identifying the resource.
  -h, --help                           help for kubectl-profefe
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
  -R, --recursive                      Process the directory used in -f, --filename recursively. Useful when you want to manage related manifests organized within the same directory. (default true)
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -l, --selector string                Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)
  -s, --server string                  The address and port of the Kubernetes API server
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use

Use "kubectl-profefe [command] --help" for more information about a command.
```

Deploy InfluxDB v2 pod, it is a test one. That's the one we will take profile from

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: influxdb-v2
  annotations:
    "profefe.com/enable": "true"
    "profefe.com/port": "9999"
spec:
  containers:
  - name: influxdb
    image: quay.io/influxdb/influxdb:2.0.0-alpha
    ports:
    - containerPort: 9999
```

Now you can capture the profiles:

```bash
kubectl profefe capture influxdb-v2
```

The profiles are stored inside your `/tmp` directory (you can change it with
`--output-dir`), so you can read it with `go tool pprof`:

```bash
go tool pprof /tmp/profile-goroutine-influxdb-v2-1575552135.pb.gz
```

If you have a profefe up and running you can push your profiles there other than
locally:

```bash
kubectl profefe capture influxdb-v2 --profefe-hostport http://localhost:10100
```

If you used kind as explained above, in order to have profefe reachable you have
to use port forwarding:

```bash
$ kubectl port-forward -n profefe svc/profefe-collector 10100
```
