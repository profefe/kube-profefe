+++
title = "Getting Started with kprofefe"
description = "Deploy kprofefe collector in Kubernetes"
weight = 15
draft = false
toc = true
bref = ""
+++

## Goal

The end goal here is to deploy a continuous profiling infrastructure in
kubernetes, and to get profiles from our pod into
[profefe](https://github.com/profefe/profefe) via kprofefe.

## Prerequisites

1. Have a Kubernetes cluster up and running
2. Deploy profefe inside or outside your Kubernetes cluster. But it has to be
   reachable from the Kubernetes cluster network. I will assume that the URL for
   it is `https://profefe.internal.company.com:10100`
3. You should have the kubectl profefe plugin already installed (checkout the
   installation doc)

## Getting Started

### Find a candidate

We have to find a good first candidate, a pod that runs an application that
exposes the pprof handler. I will assume it exposes it on port `9999`. If you do
not have anything in your Kubernetes cluster you can deploy this application as
a test

```
apiVersion: v1
kind: Pod
metadata:
  name: influxdb-v2
spec:
  containers:
  - name: influxdb
    image: quay.io/influxdb/influxdb:2.0.0-alpha
    ports:
    - containerPort: 9999
```

Now that we have the right candidate we should modify the pod adding  two
annotations:

```
  annotations:
    "profefe.com/enable": "true"
    "profefe.com/port": "9999"
```

The first one tells kprofefe that this pod can be scraped. By default kprofefe
looks for the port `6060`, with the second annotation we are overriding the
default configuration, because our application exposes profefe to port
`9999`.

### Deploy kprofefe

kprofefe can be deployed as a cronjob in Kubernetes. The one we are deploying
now will scrape codes with the right annotations in all namespaces, every 10
minutes:

```
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: kprofefe-allnamespaces
  namespace: default
spec:
  concurrencyPolicy: Replace
  failedJobsHistoryLimit: 1
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - args:
            # This cronjob will scrape all the pods that has the right
            # annotations across all the namespaces
            - --all-namespaces
            # This url represents the profefe API location.
            - --profefe-hostport
            - https://profefe.internal.company.com:10100
            image: profefe/kprofefe
            imagePullPolicy: IfNotPresent
            name: kprofefe
  schedule: '*/10 * * * *'
  successfulJobsHistoryLimit: 3
```
