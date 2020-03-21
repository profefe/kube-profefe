+++
draft= false
title = "FAQ"
description = "Asked and answered"
+++

## Do I need kube-profefe?

Kube Profefe is the official way to do continuous profiling on Kubernetes with
Profefe. If you are running profefe in your kubernetes cluster it is an easy way
to do collection without having to hack bash scripts or things like that.

## How to scale the profile gathering

If you can't run only one kprofefe for all the namespaces because you have too
many pods you can deploy more kprofefe cronjob partitioning them per namespace
or via label selector.
