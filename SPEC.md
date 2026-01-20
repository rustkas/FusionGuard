# FusionGuard Specification

This document codifies the MVP and roadmap for the <<FusionGuard>> disruption-risk platform. It mirrors the requirements provided in the request, capturing contracts, configs, testing expectations, and non-functional goals.

## Surface Overview

1. **Context**: Disruption is a costly failure in tokamaks. FusionGuard ingests telemetry, predicts disruption risk at short horizons (e.g., 50 ms / 200 ms), explains drivers, and surfaces rule-based recommendations for operators.
2. **MVP Pillars**:
   - Offline trainer produces features, calibrated model, and metadata.
   - Go services stream telemetry, compute features incrementally, run inference, and expose REST/WS APIs.
   - Observability: health, latency, metrics; reproducibility via REPORT metadata and consistent configs.
3. **Stack**: Go (streaming & API), Python (trainer), NATS (messaging), Postgres/MinIO for storage, Prometheus/Grafana, ONNX for inference. Services are independent modules with dedicated configs.

## Contracts

* gRPC proto definitions under `proto/telemetry/v1` detail ingest, feature, risk, and admin services.
* OpenAPI spec in `api/openapi.yaml` exposes REST endpoints for shots, series, events, and explanations.

## Config Strategy

Configs live under `configs/{dev,prod}` with YAML per service. They define service endpoints, NATS subjects, sampling/resample rules, model paths, and recommendation rules.

## Services Structure

Services follow this pattern:

```
services/<name>/cmd/<name>/main.go
services/<name>/internal/...        # config, business logic, health, metrics
```

Each service runs a simple HTTP health server plus metrics and loads configuration at startup.

## Trainer

`trainer/` contains Python package `fusionguard_trainer` responsible for dataset construction, labeling, training, calibration, ONNX export, and reporting. Tests ensure labeling consistency and export integrity.

## Deployment

`deploy/docker-compose.yml` orchestrates the stack: NATS, Postgres, MinIO, Prometheus, Grafana, plus all Go/Python services. `deploy/prometheus` and `deploy/grafana` provide observability defaults.

## Scripts & Automation

* `scripts/replay_shot.sh`: replay telemetry file through the ingestion pipeline.
* `scripts/loadgen.sh`: synthetic telemetry load generator.
* `scripts/lint.sh`, `scripts/test.sh`: convenience wrappers for Go/Python tooling.

## DoD (MVP)

1. Streaming ingestion → feature → inference pipeline ties together via NATS subjects and risk series.
2. API exposes health, shots, risk series, explanations, recommendations, and metrics.
3. Trainer exports risk model + calibration; inference service calibrates and provides model metadata.
4. `docker-compose up` launches full dev stack; `REPORT.md` documents metrics, latency, dataset splits.
