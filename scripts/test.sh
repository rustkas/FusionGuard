#!/usr/bin/env bash
set -euo pipefail

echo "Running go test..."
for mod in services/ingestor services/feature_service services/inference_service services/api_gateway; do
  (cd "$mod" && go test ./...)
done

echo "Running pytest against trainer..."
(cd trainer && pytest)
