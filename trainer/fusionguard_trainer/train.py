from __future__ import annotations

import argparse
import json
import subprocess
from pathlib import Path

import pandas as pd
from sklearn.linear_model import LogisticRegression
from sklearn.model_selection import train_test_split

try:
    from catboost import CatBoostClassifier
    HAS_CATBOOST = True
except ImportError:
    HAS_CATBOOST = False

try:
    from lightgbm import LGBMClassifier
    HAS_LIGHTGBM = True
except ImportError:
    HAS_LIGHTGBM = False

from .calibrate import isotonic_calibrate, platt_calibrate
from .dataset import build_dataset
from .eval import evaluate
from .export_onnx import export_model
from .features import compute_all_features
from .labeling import label_windows
from .reports import write_report


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Train FusionGuard disruption model")
    parser.add_argument("--output", type=Path, default=Path("deploy/models/dev"))
    parser.add_argument("--data", type=Path, nargs="+", help="Data file paths (CSV, HDF5, NetCDF)")
    parser.add_argument("--synthetic", action="store_true", help="Use synthetic data")
    parser.add_argument("--synthetic-shots", type=int, default=50, help="Number of synthetic shots")
    parser.add_argument("--synthetic-duration-ms", type=int, default=5000, help="Duration of synthetic shots in ms")
    parser.add_argument("--model", type=str, default="logistic", choices=["logistic", "catboost", "lightgbm"], help="Model type")
    parser.add_argument("--calibration", type=str, default="platt", choices=["platt", "isotonic", "none"], help="Calibration method")
    parser.add_argument("--horizons", type=int, nargs="+", default=[50, 200], help="Prediction horizons in ms")
    parser.add_argument("--test-split", type=float, default=0.2, help="Test set fraction")
    parser.add_argument("--seed", type=int, default=42, help="Random seed")
    return parser.parse_args()


def main() -> None:
    args = parse_args()

    # Build dataset from provided sources
    synthetic_config = None
    if args.synthetic or (not args.data):
        # Use synthetic data if explicitly requested or no data files provided
        synthetic_config = {
            "n_shots": args.synthetic_shots,
            "duration_ms": args.synthetic_duration_ms,
            "sample_rate_hz": 1000,
            "disruption_probability": 0.3,
            "seed": 42,
        }
    
    df = build_dataset(
        filepaths=args.data,
        synthetic=synthetic_config,
    )
    
    # Compute features
    channels = ["ip", "ne", "dwdt", "prad", "h_alpha"]
    available_channels = [ch for ch in channels if ch in df.columns]
    df = compute_all_features(df, channels=available_channels, windows_ms=args.horizons)
    
    # Label windows
    df = label_windows(df, args.horizons)
    
    # Select feature columns (exclude metadata)
    exclude_cols = {"shot_id", "time_unix_ns", "time_to_disruption", "disruption_time_unix_ns", "time_ms", "time"}
    exclude_cols.update({f"label_h{h}" for h in args.horizons})
    feature_cols = [col for col in df.columns if col not in exclude_cols]
    
    # Train model for each horizon
    for horizon in args.horizons:
        label_col = f"label_h{horizon}"
        if label_col not in df.columns:
            continue
        
        print(f"\nTraining model for horizon {horizon}ms...")
        
        # Prepare data
        X = df[feature_cols].fillna(0)
        y = df[label_col]
        
        # Split by shot_id to avoid data leakage
        if "shot_id" in df.columns:
            unique_shots = df["shot_id"].unique()
            train_shots, test_shots = train_test_split(unique_shots, test_size=args.test_split, random_state=args.seed)
            train_mask = df["shot_id"].isin(train_shots)
            test_mask = df["shot_id"].isin(test_shots)
            
            X_train, X_test = X[train_mask], X[test_mask]
            y_train, y_test = y[train_mask], y[test_mask]
        else:
            X_train, X_test, y_train, y_test = train_test_split(
                X, y, test_size=args.test_split, random_state=args.seed, stratify=y
            )
        
        # Train model
        if args.model == "logistic":
            model = LogisticRegression(solver="liblinear", random_state=args.seed, max_iter=1000)
            model.fit(X_train, y_train)
            
            # Get predictions
            if hasattr(model, "decision_function"):
                scores_train = model.decision_function(X_train)
                scores_test = model.decision_function(X_test)
            else:
                scores_train = model.predict_proba(X_train)[:, 1]
                scores_test = model.predict_proba(X_test)[:, 1]
            
            # Export model
            model_params = {
                "coefficients": dict(zip(feature_cols, model.coef_[0])),
                "intercept": float(model.intercept_[0]),
                "version": f"dev_h{horizon}",
            }
            
        elif args.model == "catboost" and HAS_CATBOOST:
            model = CatBoostClassifier(iterations=100, random_seed=args.seed, verbose=False)
            model.fit(X_train, y_train)
            scores_train = model.predict_proba(X_train)[:, 1]
            scores_test = model.predict_proba(X_test)[:, 1]
            
            # For CatBoost, we'll use feature importance as coefficients (simplified)
            importances = model.get_feature_importance()
            model_params = {
                "coefficients": dict(zip(feature_cols, importances.tolist())),
                "intercept": 0.0,
                "version": f"catboost_h{horizon}",
            }
            
        elif args.model == "lightgbm" and HAS_LIGHTGBM:
            model = LGBMClassifier(n_estimators=100, random_state=args.seed, verbose=-1)
            model.fit(X_train, y_train)
            scores_train = model.predict_proba(X_train)[:, 1]
            scores_test = model.predict_proba(X_test)[:, 1]
            
            # For LightGBM, use feature importance
            importances = model.feature_importances_
            model_params = {
                "coefficients": dict(zip(feature_cols, importances.tolist())),
                "intercept": 0.0,
                "version": f"lightgbm_h{horizon}",
            }
        else:
            print(f"Model {args.model} not available, falling back to logistic regression")
            model = LogisticRegression(solver="liblinear", random_state=args.seed, max_iter=1000)
            model.fit(X_train, y_train)
            scores_train = model.decision_function(X_train)
            scores_test = model.decision_function(X_test)
            model_params = {
                "coefficients": dict(zip(feature_cols, model.coef_[0])),
                "intercept": float(model.intercept_[0]),
                "version": f"dev_h{horizon}",
            }
        
        # Calibrate
        if args.calibration == "platt":
            calibrated_test, calib_params = platt_calibrate(scores_test.tolist(), y_test.tolist())
        elif args.calibration == "isotonic":
            calibrated_test, calib_params = isotonic_calibrate(scores_test.tolist(), y_test.tolist())
        else:
            calibrated_test = scores_test.tolist()
            calib_params = None
        
        # Evaluate
        time_to_disruption = None
        if "time_to_disruption" in df.columns:
            time_to_disruption = df.loc[test_mask if "shot_id" in df.columns else y_test.index, "time_to_disruption"].tolist()
        
        metrics = evaluate(y_test.tolist(), calibrated_test, time_to_disruption)
        
        # Save model and calibration
        output_dir = args.output / f"h{horizon}"
        output_dir.mkdir(parents=True, exist_ok=True)
        
        export_model(output_dir / "model_params.json", model_params)
        
        if calib_params:
            (output_dir / "calibration.json").write_text(
                json.dumps({
                    "kind": calib_params.kind,
                    "scale": calib_params.scale,
                    "offset": calib_params.offset,
                    "version": calib_params.version,
                }, indent=2),
                encoding="utf-8",
            )
        else:
            (output_dir / "calibration.json").write_text(
                json.dumps({"kind": "none", "scale": 1.0, "offset": 0.0, "version": "none"}, indent=2),
                encoding="utf-8",
            )
        
        (output_dir / "feature_order.json").write_text(
            json.dumps({"features": feature_cols}, indent=2),
            encoding="utf-8",
        )
        
        # Get git commit hash for reproducibility
        commit_hash = "unknown"
        try:
            result = subprocess.run(
                ["git", "rev-parse", "HEAD"],
                capture_output=True,
                text=True,
                timeout=5,
            )
            if result.returncode == 0:
                commit_hash = result.stdout.strip()
        except Exception:
            pass
        
        # Write report
        write_report(
            output_dir / "REPORT.md",
            {
                "roc_auc": metrics.roc_auc,
                "pr_auc": metrics.pr_auc,
                "recall_at_fpr_1%": metrics.recall_at_fpr,
                "mean_lead_time_ms": metrics.mean_lead_time_ms,
                "brier_score": metrics.brier_score,
                "calibration_error": metrics.calibration_error,
            },
            {
                "samples": str(len(df)),
                "train_samples": str(len(X_train)),
                "test_samples": str(len(X_test)),
                "features": str(len(feature_cols)),
                "model": args.model,
                "calibration": args.calibration,
                "horizon_ms": str(horizon),
                "commit_hash": commit_hash,
            },
        )
        
        print(f"  ROC-AUC: {metrics.roc_auc:.4f}")
        print(f"  PR-AUC: {metrics.pr_auc:.4f}")
        print(f"  Recall@FPR=1%: {metrics.recall_at_fpr:.4f}")
        print(f"  Mean lead time: {metrics.mean_lead_time_ms:.2f} ms")
    
    print("\nTraining complete")


if __name__ == "__main__":
    main()
