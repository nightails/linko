# Linko

This is a toy URL shortener project, to be used as the starter repo for the Logging and Telemetry course on [Boot.dev](https://www.boot.dev/).

It's intentionally small, a little messy, and realistic enough to practice adding logs, metrics, and traces in Go.

## Monitoring

The project uses Prometheus and Grafana for metric monitoring.

### Prometheus

Prometheus is configured to collect and store metrics. To start the Prometheus service, run:

```bash
docker compose up
```

You can access the Prometheus dashboard at [http://localhost:9090](http://localhost:9090).
