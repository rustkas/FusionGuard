from __future__ import annotations

from pathlib import Path
from typing import Any

import pandas as pd

from .loaders import load_data, generate_synthetic_data


def build_dataset(
    sources: list[dict[str, Any]] | None = None,
    filepaths: list[Path | str] | None = None,
    synthetic: dict[str, Any] | None = None,
) -> pd.DataFrame:
    """Build dataset from multiple sources.
    
    Args:
        sources: List of dict samples (legacy format)
        filepaths: List of file paths to load
        synthetic: Dict with synthetic data parameters (n_shots, duration_ms, etc.)
    
    Returns:
        Combined DataFrame with all data
    """
    dfs = []
    
    # Load from file paths
    if filepaths:
        for filepath in filepaths:
            df = load_data(filepath)
            dfs.append(df)
    
    # Generate synthetic data
    if synthetic:
        df = generate_synthetic_data(**synthetic)
        dfs.append(df)
    
    # Legacy: load from samples dict
    if sources:
        df = pd.DataFrame(sources)
        dfs.append(df)
    
    if not dfs:
        raise ValueError("No data sources provided")
    
    # Combine all dataframes
    result = pd.concat(dfs, ignore_index=True)
    
    # Ensure required columns exist
    if "time_to_disruption" not in result.columns:
        if "disruption_time_unix_ns" in result.columns and "time_unix_ns" in result.columns:
            result["time_to_disruption"] = (
                result["disruption_time_unix_ns"] - result["time_unix_ns"]
            ) / 1_000_000  # Convert to ms
        else:
            # No disruption info - mark as safe
            result["time_to_disruption"] = float("inf")
    
    return result
