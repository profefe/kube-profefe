[Profefe](https://github.com/profefe/profefe) is a project developed by
[@narqo](https://github.com/narqo). I was looking for a solution to do
continuous profiling and I realized his code was well abstracted and comfortable
to extend. The API server was already done and I decided to write an integration
with Kubernetes.

## kube-profefe

This project is a bridge between profefe and Kubernetes. At the moment it serves
two different binaries:

* `kubectl-profefe` a kubectl plugin that helps you to `caputre` pprof profiles,
  storing them locally or in pprofefe. It uses `port-forwarding` to expose the
  pprof port locally.
* `kprofefe` is a cli that you can run as a cronjob in your kubernetes cluster.
  It discovers running pods in your clusters, it downloads profiles and it
  pushes them in profefe.

NB: if your configuration does not allow you to do `kubectl port-forward` the
`kubectl` plugin will not work.

### How it works

Golang has an http handler that exposes pprof over http, via annotation we can
specify if a pod has profiles to capture and with other annotations we can
configure path and port.

The annotations are:

* `profefe.com/enable=true` is the annotation that tells kube-profefe to capture
  profiles from that pod.
* `profefe.com/port=8085` tells kube-profefe where to look for a pprof http
  server. By default it is 6060.
* `profefe.com/path=/debug/pprof` tells kube-profefe where to look for a pprof http
  server. By default it is `/debug/pprof`.

### Getting Started with kubectl-profefe

Start minikube and deploy this pod:

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

```
go tool pprof /tmp/profile-goroutine-influxdb-v2-1575552135.pb.gz
```

If you have a profefe up and running you can push your profiles there other than
locally:

```
kubectl profefe capture influxdb-v2 --profefe-hostport http://localhost:10100
```
