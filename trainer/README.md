# FusionGuard Trainer

Run the trainer locally with:

```
python -m fusionguard_trainer.train --output deploy/models/dev
```

This builds a toy dataset, trains a logistic model, calibrates scores, and writes `model.onnx`, `calibration.json`, `feature_order.json`, and `REPORT.md`.

Use `pytest tests/` to verify helper utilities.
