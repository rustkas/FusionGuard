from __future__ import annotations

from pathlib import Path


def write_report(path: Path, metrics: dict[str, float], dataset_info: dict[str, str]) -> None:
    """Write training report with metrics and dataset information."""
    lines = ["# FusionGuard Training Report", ""]
    
    lines.append("## Metrics")
    for k, v in metrics.items():
        if isinstance(v, float):
            lines.append(f"- **{k}**: {v:.4f}")
        else:
            lines.append(f"- **{k}**: {v}")
    lines.append("")
    
    lines.append("## Dataset Information")
    for k, v in dataset_info.items():
        lines.append(f"- **{k}**: {v}")
    lines.append("")
    
    lines.append("## Notes")
    lines.append("- Metrics computed on test set")
    lines.append("- Lead time is measured from when prediction crosses threshold to actual disruption")
    lines.append("- Calibration error (ECE) measures how well-calibrated probabilities are")
    
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")
