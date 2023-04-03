fastbreaker
===========

[![Go Reference](https://pkg.go.dev/badge/github.com/bluekiri/fastbreaker/prometheus.svg)](https://pkg.go.dev/github.com/bluekiri/fastbreaker/prometheus) [![CI](https://github.com/bluekiri/fastbreaker/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/bluekiri/fastbreaker/actions/workflows/ci.yml)

[fastbreaker.prometheus](https://github.com/bluekiri/fastbreaker/prometheus) simplifies regitering metrics generated from [fastbreaker](https://github.com/bluekiri/fastbreaker) into [prometheus](https://prometheus.io/).

Installation
------------

```
go get github.com/bluekiri/fastbreaker/prometheus
```

Usage
-----

The function `prometheus.RegisterMetricsToDefaultRegisterer` registers the `fastbreaker.FastBreaker` metrics with the default Registerer.

The function `prometheus.RegisterMetrics` registers the `fastbreaker.FastBreaker` metrics with the provided Registerer.

The function `RegisterMetricsWithFactory` registers the `fastbreaker.FastBreaker` metrics with the provided `promauto.Factory`.

All three functions return an error if the circuit breaker name is not a valid UTF-8 string.

Example
-------

```go
var cb fastbreaker.FastBreaker

prometheus.RegisterMetricsToDefaultRegisterer("my-circuit-breaker", cb)
```

License
-------

The MIT License (MIT)

See [LICENSE](https://github.com/bluekiri/fastbreaker/blob/master/LICENSE) for details.
