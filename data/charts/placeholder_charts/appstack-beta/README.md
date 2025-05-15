# appstack-beta

Beta application stack with API and worker.

This is an umbrella chart.
Subcharts (located in the `./charts/` directory):
- api-service-beta
- worker-beta

This chart version uses `@{variable.path}` placeholders for values sourced from external configuration.

See `values.yaml` (and `values-tags.yaml` if present for umbrella charts) for configuration options.
