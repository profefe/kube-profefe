+++
title = "Introduction"
description = "Let's sync up on some concepts"
weight = 10
draft = false
toc = true
bref = ""
+++

## Introduction

This project is a bridge between profefe and Kubernetes. At the moment it serves
two different binaries:

* `kubectl-profefe` a kubectl plugin that helps you to `capture` pprof profiles,
  storing them locally or in profefe. It uses `port-forwarding` to expose the
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
* `profefe.com/service=frontend` tells kube-profefe the name of the service
  usable to lookup the profile. If the annotation is not specified it uses the
  pod name. My suggestion is to set this annotation because pods are ephemeral
  and lookup by pod name may be hard to do.
* `profefe.com/path=/debug/pprof` tells kube-profefe where to look for a pprof http
  server. By default it is `/debug/pprof`.
