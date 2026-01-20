# FusionGuard

FusionGuard is a real-time disruption risk prediction and guidance platform for tokamak plasma control. It combines streaming ingestion, incremental feature computation, calibrated inference, rule-based recommendations, and a lightweight UI/API stack to demonstrate a production-grade warning system built on open data.

## FusionGuard — Real-Time Disruption Prediction and Plasma Control Recommendation System

### Overview

**FusionGuard** is an end-to-end real-time system for **early disruption prediction in tokamak plasma discharges**, with explainability and actionable control recommendations.

A plasma disruption is an unplanned termination of a discharge that can damage equipment, interrupt experiments, and significantly increase operational costs. Preventing disruptions is one of the most critical engineering challenges in magnetic confinement fusion.

FusionGuard addresses this problem by combining **streaming data processing, low-latency inference, explainable machine learning, and production-grade system design**.

The system continuously ingests plasma telemetry, predicts disruption risk on short time horizons (tens to hundreds of milliseconds), explains which signals drive the risk increase, and suggests high-level mitigation actions.

This project is designed to demonstrate not just a model, but a **reliable, observable, reproducible production system** — the kind expected in real fusion control environments.

### Key Capabilities

#### Real-Time Disruption Risk Prediction

* Continuous ingestion of multi-channel plasma telemetry
* Online feature computation in sliding time windows
* Low-latency inference with multiple prediction horizons (e.g. 50 ms, 200 ms)
* Calibrated probabilistic output (`risk ∈ [0, 1]`)

#### Explainability

* Identification of dominant signals contributing to rising risk
* Offline SHAP-based analysis for trained models
* Lightweight online “top drivers” explanation suitable for real-time use

#### Action Recommendations (MVP)

* Rule-based mitigation suggestions layered on top of risk and signal patterns
* Output as high-level action categories (e.g. heating reduction, density correction)
* Confidence and rationale attached to each recommendation
* Fully configurable and unit-tested rule engine

#### Observability & Reliability

* End-to-end latency tracking (telemetry → risk)
* Metrics for throughput, dropped samples, model versioning
* Graceful degradation under missing or partial data
* Prometheus-compatible metrics and health checks

#### Reproducible ML Pipeline

* Offline training on historical discharges from public datasets
* Strict experiment tracking:

  * code version (commit hash)
  * dataset split
  * model parameters
  * evaluation metrics
* Versioned model artifacts published to object storage

### Intended Users

* **Plasma operators**
  Monitor real-time disruption risk, understand causes, and receive mitigation guidance during a discharge.

* **ML / control engineers**
  Analyze model quality, latency, feature behavior, drift, and retraining needs.

* **Researchers**
  Reproduce experiments, compare models, and export structured disruption events for further study.

### Typical Usage Scenarios

* *During a discharge:*
  “I see the disruption risk rising in real time, with alerts when thresholds are crossed.”

* *After a discharge:*
  “I can inspect when degradation started, which signals were responsible, and how early the system reacted.”

* *Model comparison:*
  “I can compare different model versions by accuracy, lead time, latency, and stability.”

### Architecture (High-Level)

FusionGuard is intentionally split into clear, production-ready components:

* **Ingestion service**
  Validates, buffers, resamples, and streams telemetry data.

* **Feature service**
  Maintains sliding windows and computes features incrementally with O(1) updates.

* **Inference service**
  Applies trained models, produces risk estimates, explanations, and events.

* **API / UI layer**
  Exposes data via HTTP/gRPC and provides a minimal web dashboard.

* **Training pipeline (Python)**
  Prepares datasets, trains models, evaluates metrics, and publishes artifacts.

The stack emphasizes **low latency, stability, and observability**, using Go or Rust for real-time paths and Python for ML training.

### Quality Metrics

FusionGuard reports metrics that are meaningful both technically and operationally:

* PR-AUC / ROC-AUC per prediction horizon
* Recall at fixed false positive rate (e.g. FPR = 1%)
* Mean lead time before disruption
* Calibration error (ECE, Brier score)
* Online p95 latency and sample drop rate

### Why This Project Matters

FusionGuard is compelling because it demonstrates:

* A **real industrial safety-critical problem**, not a toy task
* A full **end-to-end system**, not just a notebook with a model
* Clear, measurable KPIs (latency, lead time, recall@FPR)
* Strong alignment with modern fusion, ML, and control engineering needs
* Production-grade thinking: reliability, observability, reproducibility

This is exactly the type of system fusion laboratories and deep-tech startups are actively trying to build — and struggling to staff with engineers who can deliver it end to end.

## Highlights

- **Streaming pipeline**: Go services ingest telemetry over HTTP, compute features incrementally, and emit risk scores via NATS.
- **Database integration**: PostgreSQL storage for shots, telemetry, risks, and events with full query API.
- **REST API**: Complete OpenAPI-compliant API for accessing shots, time series, events, and explanations.
- **Trainer pipeline**: Python tooling supports CSV, HDF5, NetCDF formats, synthetic data generation, and trains models (LogisticRegression, CatBoost, LightGBM) with comprehensive metrics.
- **Web UI**: Minimal dashboard for visualizing risk over time, top drivers, recommendations, and events.
- **Observability-ready**: Health checks, Prometheus metrics per service; Docker Compose brings up Prometheus + Grafana.
- **Reproducibility**: Trainer writes REPORT.md with metrics, dataset splits, commit hash, and model metadata.

## Getting Started

### Prerequisites

- Go 1.21+
- Python 3.11+
- Docker and Docker Compose
- PostgreSQL client (optional, for direct DB access)

### Quick Start

1. **Start infrastructure services:**
   ```bash
   cd deploy
   docker-compose up -d postgres nats prometheus grafana
   ```

2. **Initialize database:**
   ```bash
   # Run SQL migrations
   docker-compose exec postgres psql -U fusion -d fusionguard -f /path/to/deploy/sql/001_init.sql
   ```

3. **Train a model (optional, uses synthetic data by default):**
   ```bash
   cd trainer
   python3 -m fusionguard_trainer.train --synthetic --synthetic-shots 50
   ```

4. **Start services:**
   ```bash
   # In separate terminals or use docker-compose
   cd services/ingestor && go run cmd/ingestor/main.go -config ../../configs/dev/ingestor.yaml
   cd services/feature_service && go run cmd/feature_service/main.go -config ../../configs/dev/features.yaml
   cd services/inference_service && go run cmd/inference_service/main.go -config ../../configs/dev/inference_service.yaml
   cd services/api_gateway && go run cmd/api_gateway/main.go -config ../../configs/dev/api_gateway.yaml
   ```

5. **Generate and replay data:**
   ```bash
   # Generate synthetic load
   ./scripts/loadgen.sh --duration-sec 60 --rate-hz 1000
   
   # Or replay from a file
   ./scripts/replay_shot.sh data/sample_shot.csv
   ```

6. **Access the UI:**
   Open `http://localhost:8080` in your browser to view the FusionGuard dashboard.

### Using Docker Compose (Full Stack)

```bash
cd deploy
docker-compose up
```

This starts all services including the Go services. Make sure to build the services first or update docker-compose.yml with proper build contexts.

## Repository Layout

```
fusionguard/
  README.md              # This file
  SPEC.md               # Detailed specification
  REPORT.md             # Training report (generated)
  api/
    openapi.yaml        # API specification
  configs/
    dev/                # Development configurations
    prod/               # Production configurations
  deploy/
    docker-compose.yml  # Full stack orchestration
    sql/                # Database migrations
    prometheus/         # Prometheus config
    grafana/            # Grafana dashboards
  pkg/
    storage/            # Database abstraction layer
    telemetry/          # Telemetry data types
  proto/                # gRPC protocol definitions
  scripts/
    replay_shot.sh      # Replay telemetry from file
    loadgen.sh          # Generate synthetic load
  services/
    ingestor/           # Telemetry ingestion service
    feature_service/    # Feature computation service
    inference_service/  # Risk prediction service
    api_gateway/        # REST API and UI server
  trainer/              # Python training pipeline
    fusionguard_trainer/
      train.py          # Main training script
      dataset.py        # Dataset construction
      features.py       # Feature engineering
      loaders.py        # Data loaders (CSV, HDF5, NetCDF)
      eval.py           # Evaluation metrics
      calibrate.py      # Probability calibration
  ui/                   # Web UI
    index.html
    static/
      js/app.js
      css/style.css
```

## API Endpoints

- `GET /shots` - List all shots
- `GET /shots/{shot_id}/series?kind=risk|telemetry|features&from_unix_ns=...&to_unix_ns=...` - Get time series
- `GET /shots/{shot_id}/events` - Get events (alerts, disruptions)
- `GET /shots/{shot_id}/explain?at_unix_ns=...` - Get explanation at specific time
- `POST /ingest` (ingestor) - Ingest telemetry point

See `api/openapi.yaml` for full API documentation.

## Training Models

The trainer supports multiple data formats and model types:

```bash
# Train with synthetic data
python3 -m fusionguard_trainer.train --synthetic --synthetic-shots 100

# Train with CSV file
python3 -m fusionguard_trainer.train --data data/shots.csv

# Train with CatBoost
python3 -m fusionguard_trainer.train --model catboost --synthetic

# Train with isotonic calibration
python3 -m fusionguard_trainer.train --calibration isotonic --synthetic
```

Models are saved to `deploy/models/dev/` by default. Each horizon (50ms, 200ms) gets its own model directory.

## Configuration

All services use YAML configuration files in `configs/dev/` or `configs/prod/`. Key settings:

- **Ingestor**: NATS URL, allowed channels, storage DSN
- **Feature Service**: Window sizes, channel list, feature types
- **Inference Service**: Model path, calibration path, thresholds, rules
- **API Gateway**: Storage DSN, cache settings

## Development

### Running Tests

```bash
# Go tests
cd services/ingestor && go test ./...
cd services/feature_service && go test ./...
cd services/inference_service && go test ./...
cd services/api_gateway && go test ./...

# Python tests
cd trainer && python3 -m pytest tests/
```

### Building Services

```bash
# Each service has its own go.mod
cd services/ingestor && go build ./cmd/ingestor
cd services/feature_service && go build ./cmd/feature_service
cd services/inference_service && go build ./cmd/inference_service
cd services/api_gateway && go build ./cmd/api_gateway
```

## License

Apache-2.0
