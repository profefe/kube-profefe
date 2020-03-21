+++
title = "Installation"
description = "Setting up the tools"
weight = 20
draft = false
toc = true
bref = ""
+++
This project is made of two components. A kubectl plugin and the collector.

You can download both of them from the [release
page](https://github.com/profefe/kube-profefe/releases) via GitHub.

The kubectl plugin usually runs from your laptop, and it is built for multiple
platforms: Linux, Mac.

The collector called `kprofefe` usually runs as a cronjob in Kubernetes. You can
find an example of it in `./contrib/kubernetes/kprofefe.yaml`.

The one reported there will scrape all the pods (with the right annotations)
every 10 minutes because it runs with the flags:

```yaml
  containers:
  - args:
    - --all-namespaces
```

`--all-namespaces` means that it will look for all the namespace one by one.

You can change it with the well known filters:

- `-n` to change the namespace
- `-l` to filter by labels

More about label selector in the [Kubernetes
site](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors)
