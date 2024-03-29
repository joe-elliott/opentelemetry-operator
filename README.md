[![Continuous Integration][github-workflow-img]][github-workflow] [![Go Report Card][goreport-img]][goreport] [![GoDoc][godoc-img]][godoc] [![Maintainability][code-climate-img]][code-climate] [![codecov][codecov-img]][codecov]

# OpenTelemetry Operator for Kubernetes

The OpenTelemetry Operator is an implementation of a [Kubernetes Operator](https://coreos.com/operators/).

At this point, it has [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-service) as the only managed component.

## Getting started

To install the operator, run:
```
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/crds/opentelemetry.io_opentelemetrycollectors_crd.yaml
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/service_account.yaml
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/role.yaml
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/role_binding.yaml
kubectl create -f https://raw.githubusercontent.com/jpkrohling/opentelemetry-operator/master/deploy/operator.yaml
```

Once the `opentelemetry-operator` deployment is ready, create an OpenTelemetry Collector (otelcol) instance, like:

```console
$ kubectl apply -f - <<EOF
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: simplest
spec:
  config: |
    receivers:
      jaeger:

    processors:
      queued-retry:

    exporters:
      logging:

    pipelines:
      traces:
        receivers: [jaeger]
        processors: [queued-retry]
        exporters: [logging]
EOF
```

This will create an OpenTelemetry Collector instance named `simplest`, exposing a `jaeger-grpc` port to consume spans from your instrumented applications and exporting those spans via `logging`, which writes the spans to the console (`stdout`) of the OpenTelemetry Collector instance that receives the span.

The `config` node holds the `YAML` that should be passed down as-is to the underlying OpenTelemetry Collector instances. Refer to the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-service) documentation for a reference of the possible entries.

At this point, the Operator does *not* validate the contents of the configuration file: if the configuration is invalid, the instance will still be created but the underlying OpenTelemetry Collector might crash.

### Deployment modes

The `CustomResource` for the `OpenTelemetryCollector` exposes a property named `.Spec.Mode`, which can be used to specify whether the collector should run as a `DaemonSet` or as a `Deployment` (default). Look at the `examples/daemonset.yaml` for reference.

## Prometheus ServiceMonitor objects

When the Prometheus Operator is available in the same cluster as the OpenTelemetry Operator, the OpenTelemetry Operator will automatically create the relevant `ServiceMonitor` objects:

* One set for the OpenTelemetry Operator itself
* One set for each managed OpenTelemetry instance

Refer to the Prometheus Operator for complete instructions on how to do a production-quality installation. For development purposes, the following will do:

```console
$ kubectl apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/v0.33.0/bundle.yaml
```

When deploying the example `simplest.yaml`, the following `ServiceMonitor` will be created once the Prometheus Operator is available:

```console
$ kubectl get servicemonitors simplest-collector
NAME                 AGE
simplest-collector   103s
```

## Contributing and Developing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## License
  
[Apache 2.0 License](./LICENSE).

[github-workflow]: https://github.com/jpkrohling/opentelemetry-operator/actions
[github-workflow-img]: https://github.com/jpkrohling/opentelemetry-operator/workflows/Continuous%20Integration/badge.svg
[goreport-img]: https://goreportcard.com/badge/github.com/jpkrohling/opentelemetry-operator
[goreport]: https://goreportcard.com/report/github.com/jpkrohling/opentelemetry-operator
[godoc-img]: https://godoc.org/github.com/jpkrohling/opentelemetry-operator?status.svg
[godoc]: https://godoc.org/github.com/jpkrohling/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1#OpenTelemetryCollector
[code-climate]: https://codeclimate.com/github/jpkrohling/opentelemetry-operator/maintainability
[code-climate-img]: https://api.codeclimate.com/v1/badges/7bb215eea77fc9c24484/maintainability
[codecov]: https://codecov.io/gh/jpkrohling/opentelemetry-operator
[codecov-img]: https://codecov.io/gh/jpkrohling/opentelemetry-operator/branch/master/graph/badge.svg
