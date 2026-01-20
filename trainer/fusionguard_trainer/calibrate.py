from __future__ import annotations

from dataclasses import dataclass

import numpy as np
from sklearn.calibration import CalibratedClassifierCV
from sklearn.isotonic import IsotonicRegression
from sklearn.linear_model import LogisticRegression


@dataclass
class CalibrationParams:
    kind: str  # "platt", "isotonic", "none"
    scale: float
    offset: float
    version: str


def platt_calibrate(
    scores: list[float],
    y_true: list[int] | None = None,
) -> tuple[list[float], CalibrationParams]:
    """Apply Platt scaling calibration.
    
    If y_true is provided, fits calibration on data.
    Otherwise, uses simple normalization.
    """
    scores_array = np.array(scores)
    
    if y_true is not None:
        # Fit Platt scaling
        y_true_array = np.array(y_true)
        lr = LogisticRegression(solver="liblinear")
        lr.fit(scores_array.reshape(-1, 1), y_true_array)
        
        # Apply calibration
        calibrated = lr.predict_proba(scores_array.reshape(-1, 1))[:, 1]
        
        params = CalibrationParams(
            kind="platt",
            scale=lr.coef_[0][0],
            offset=lr.intercept_[0],
            version="fitted",
        )
    else:
        # Simple normalization
        min_score = np.min(scores_array)
        max_score = np.max(scores_array)
        range_score = max_score - min_score
        
        if range_score > 1e-9:
            normalized = (scores_array - min_score) / range_score
        else:
            normalized = scores_array
        
        # Apply sigmoid
        calibrated = 1.0 / (1.0 + np.exp(-normalized))
        
        params = CalibrationParams(
            kind="platt",
            scale=1.0,
            offset=0.0,
            version="normalized",
        )
    
    return calibrated.tolist(), params


def isotonic_calibrate(
    scores: list[float],
    y_true: list[int],
) -> tuple[list[float], CalibrationParams]:
    """Apply isotonic regression calibration."""
    scores_array = np.array(scores)
    y_true_array = np.array(y_true)
    
    iso = IsotonicRegression(out_of_bounds="clip")
    calibrated = iso.fit_transform(scores_array, y_true_array)
    
    params = CalibrationParams(
        kind="isotonic",
        scale=1.0,
        offset=0.0,
        version="fitted",
    )
    
    return calibrated.tolist(), params

