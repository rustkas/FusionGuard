from __future__ import annotations

from dataclasses import dataclass

import numpy as np
from sklearn.metrics import (
    average_precision_score,
    brier_score_loss,
    precision_recall_curve,
    roc_auc_score,
    roc_curve,
)


@dataclass
class Metrics:
    roc_auc: float
    pr_auc: float
    recall_at_fpr: float  # Recall at fixed FPR (e.g., 1%)
    mean_lead_time_ms: float  # Mean lead time before disruption
    brier_score: float
    calibration_error: float  # Expected Calibration Error (ECE)


def recall_at_fixed_fpr(y_true: list[int], y_pred: list[float], target_fpr: float = 0.01) -> float:
    """Compute recall at a fixed false positive rate."""
    fpr, tpr, thresholds = roc_curve(y_true, y_pred)
    
    # Find threshold closest to target FPR
    idx = np.argmin(np.abs(fpr - target_fpr))
    threshold = thresholds[idx] if idx < len(thresholds) else 0.5
    
    # Compute recall at this threshold
    y_pred_binary = (np.array(y_pred) >= threshold).astype(int)
    tp = np.sum((np.array(y_true) == 1) & (y_pred_binary == 1))
    fn = np.sum((np.array(y_true) == 1) & (y_pred_binary == 0))
    
    recall = tp / max(1, tp + fn)
    return recall


def mean_lead_time(
    y_true: list[int],
    y_pred: list[float],
    time_to_disruption: list[float],
    threshold: float = 0.5,
) -> float:
    """Compute mean lead time (time before disruption when prediction crosses threshold)."""
    y_pred_array = np.array(y_pred)
    y_true_array = np.array(y_true)
    time_array = np.array(time_to_disruption)
    
    # Find indices where prediction crosses threshold before disruption
    lead_times = []
    for i in range(len(y_pred_array)):
        if y_true_array[i] == 1 and time_array[i] < np.inf:
            # Find when prediction first crossed threshold
            # Look backwards from disruption
            for j in range(i, -1, -1):
                if y_pred_array[j] >= threshold:
                    lead_time = time_array[i] - time_array[j]
                    if lead_time > 0:
                        lead_times.append(lead_time)
                    break
    
    return np.mean(lead_times) if lead_times else 0.0


def calibration_error(y_true: list[int], y_pred: list[float], n_bins: int = 10) -> float:
    """Compute Expected Calibration Error (ECE)."""
    y_true_array = np.array(y_true)
    y_pred_array = np.array(y_pred)
    
    bin_boundaries = np.linspace(0, 1, n_bins + 1)
    bin_lowers = bin_boundaries[:-1]
    bin_uppers = bin_boundaries[1:]
    
    ece = 0.0
    for bin_lower, bin_upper in zip(bin_lowers, bin_uppers):
        # Find predictions in this bin
        in_bin = (y_pred_array > bin_lower) & (y_pred_array <= bin_upper)
        prop_in_bin = in_bin.mean()
        
        if prop_in_bin > 0:
            # Accuracy in this bin
            accuracy_in_bin = y_true_array[in_bin].mean()
            # Average confidence in this bin
            avg_confidence_in_bin = y_pred_array[in_bin].mean()
            # Weighted difference
            ece += np.abs(avg_confidence_in_bin - accuracy_in_bin) * prop_in_bin
    
    return ece


def evaluate(
    y_true: list[int],
    y_pred: list[float],
    time_to_disruption: list[float] | None = None,
) -> Metrics:
    """Evaluate model predictions with comprehensive metrics."""
    roc_auc = roc_auc_score(y_true, y_pred)
    pr_auc = average_precision_score(y_true, y_pred)
    recall_at_fpr_001 = recall_at_fixed_fpr(y_true, y_pred, target_fpr=0.01)
    brier = brier_score_loss(y_true, y_pred)
    ece = calibration_error(y_true, y_pred)
    
    mean_lead = 0.0
    if time_to_disruption is not None:
        mean_lead = mean_lead_time(y_true, y_pred, time_to_disruption)
    
    return Metrics(
        roc_auc=roc_auc,
        pr_auc=pr_auc,
        recall_at_fpr=recall_at_fpr_001,
        mean_lead_time_ms=mean_lead,
        brier_score=brier,
        calibration_error=ece,
    )
