from __future__ import annotations

import pandas as pd


def label_windows(df: pd.DataFrame, horizons: list[int]) -> pd.DataFrame:
    """Produce binary labels for each horizon in milliseconds."""
    labels = {}
    for h in horizons:
        labels[f"label_h{h}"] = (df["time_to_disruption"] <= h).astype(int)
    return df.assign(**labels)
