+++
title = "Developing"
description = "Contributing to kube-profefe"
weight = 30
draft = false
toc = true
bref = ""
+++

This project is written in Go and it uses [go
module](https://blog.golang.org/using-go-modules) as a dependency manager.

The tool chain I use is very simple and it does not require anything more than
what Go requires.

```bash
$ go build cmd/kprofefe
$ go test ./...
$ go vet ./...
```

And so on.

## Delivery

This project is in continuous delivery and it uses
[GoReleaser](https://github.com/goreleaser/goreleaser) with GitHub Actions.

Every time a new tag is pushed the CI runs GoReleser. It updates the changelog
in the release page and it builds binaries for multiple architectures and docker
images.

Docker images are available on [Docker Hub](https://hub.docker.com/u/profefe).

Locally you can build a release anytime, even if you do not have a tag with:

```
$ goreleaser release --snapshot --rm-dist --skip-publish
```

The process will create binaries and Docker images.

## Continuous Integration

The continuous integration is managed via GitHub Actions.

## KinD

I use [kubernetes-sigs/kind](https://github.com/kubernetes-sigs/kind) where I
have to spin up a temporary kubernetes cluster for my test. It requires Docker.

```bash
$ kind create cluster
```

I can run `kprofefe` locally to see how it works but it won't be able to
actually scrape profiles because it tries to reach the pprof endpoint using POD
IPs, and they are not reachable from my local laptop.

If you would like to test the entire lifecycle you have to deploy your version
of kprofefe in Kubernetes. I do not have a workflow to share about it yet.
