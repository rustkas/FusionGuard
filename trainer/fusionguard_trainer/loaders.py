from __future__ import annotations

import json
import random
from pathlib import Path
from typing import Any

import numpy as np
import pandas as pd

try:
    import h5py
    HAS_H5PY = True
except ImportError:
    HAS_H5PY = False

try:
    import netCDF4
    HAS_NETCDF = True
except ImportError:
    HAS_NETCDF = False


def load_csv(filepath: Path | str, shot_id_column: str = "shot_id") -> pd.DataFrame:
    """Load telemetry data from CSV file.
    
    Expected columns: shot_id, time_ms (or time_unix_ns), channel names (ip, ne, dwdt, prad, h_alpha, etc.)
    """
    df = pd.read_csv(filepath)
    
    # Normalize time column
    if "time_unix_ns" in df.columns:
        time_col = "time_unix_ns"
    elif "time_ms" in df.columns:
        time_col = "time_ms"
        df["time_unix_ns"] = df["time_ms"] * 1_000_000
    elif "time" in df.columns:
        time_col = "time"
        df["time_unix_ns"] = df["time"] * 1_000_000_000  # Assume seconds
    else:
        raise ValueError("No time column found (expected time_unix_ns, time_ms, or time)")
    
    return df


def load_hdf5(filepath: Path | str, shot_id: str | None = None) -> pd.DataFrame:
    """Load telemetry data from HDF5 file.
    
    Expected structure:
    - /shots/{shot_id}/time (or time_unix_ns)
    - /shots/{shot_id}/channels/{channel_name}
    - /shots/{shot_id}/metadata/disruption_time (optional)
    """
    if not HAS_H5PY:
        raise ImportError("h5py is required for HDF5 support. Install with: pip install h5py")
    
    with h5py.File(filepath, "r") as f:
        if shot_id is None:
            # Try to find first shot
            if "shots" not in f:
                raise ValueError("No 'shots' group found in HDF5 file")
            shot_ids = list(f["shots"].keys())
            if not shot_ids:
                raise ValueError("No shots found in HDF5 file")
            shot_id = shot_ids[0]
        
        shot_group = f[f"shots/{shot_id}"]
        
        # Load time
        if "time_unix_ns" in shot_group:
            time_data = shot_group["time_unix_ns"][:]
        elif "time" in shot_group:
            time_data = shot_group["time"][:] * 1_000_000_000  # Assume seconds
        else:
            raise ValueError(f"No time data found for shot {shot_id}")
        
        # Load channels
        data = {"time_unix_ns": time_data}
        if "channels" in shot_group:
            channels_group = shot_group["channels"]
            for channel_name in channels_group.keys():
                data[channel_name] = channels_group[channel_name][:]
        
        df = pd.DataFrame(data)
        df["shot_id"] = shot_id
        
        # Load disruption time if available
        if "metadata" in shot_group and "disruption_time" in shot_group["metadata"]:
            disruption_time = shot_group["metadata"]["disruption_time"][()]
            df["disruption_time_unix_ns"] = disruption_time
            df["time_to_disruption"] = (disruption_time - df["time_unix_ns"]) / 1_000_000  # Convert to ms
        
        return df


def load_netcdf(filepath: Path | str, shot_id: str | None = None) -> pd.DataFrame:
    """Load telemetry data from NetCDF file.
    
    Expected variables:
    - time (or time_unix_ns)
    - channel variables (ip, ne, dwdt, prad, h_alpha, etc.)
    - disruption_time (optional)
    """
    if not HAS_NETCDF:
        raise ImportError("netCDF4 is required for NetCDF support. Install with: pip install netcdf4")
    
    with netCDF4.Dataset(filepath, "r") as ds:
        # Load time
        if "time_unix_ns" in ds.variables:
            time_data = ds.variables["time_unix_ns"][:]
        elif "time" in ds.variables:
            time_data = ds.variables["time"][:] * 1_000_000_000  # Assume seconds
        else:
            raise ValueError("No time variable found")
        
        data = {"time_unix_ns": time_data}
        
        # Load all other variables as channels
        for var_name in ds.variables:
            if var_name not in ["time", "time_unix_ns"]:
                data[var_name] = ds.variables[var_name][:]
        
        df = pd.DataFrame(data)
        
        # Try to get shot_id from global attributes or filename
        if shot_id is None:
            if "shot_id" in ds.ncattrs():
                shot_id = ds.getncattr("shot_id")
            else:
                shot_id = Path(filepath).stem
        
        df["shot_id"] = shot_id
        
        # Load disruption time if available
        if "disruption_time" in ds.variables:
            disruption_time = ds.variables["disruption_time"][:]
            if len(disruption_time) > 0:
                df["disruption_time_unix_ns"] = disruption_time[0]
                df["time_to_disruption"] = (disruption_time[0] - df["time_unix_ns"]) / 1_000_000
        
        return df


def generate_synthetic_data(
    n_shots: int = 10,
    duration_ms: int = 5000,
    sample_rate_hz: int = 1000,
    disruption_probability: float = 0.3,
    seed: int | None = None,
) -> pd.DataFrame:
    """Generate synthetic tokamak telemetry data for testing.
    
    Args:
        n_shots: Number of shots to generate
        duration_ms: Duration of each shot in milliseconds
        sample_rate_hz: Sampling rate in Hz
        disruption_probability: Probability that a shot will have a disruption
        seed: Random seed for reproducibility
    """
    if seed is not None:
        random.seed(seed)
        np.random.seed(seed)
    
    all_shots = []
    
    for shot_idx in range(n_shots):
        shot_id = f"synthetic_{shot_idx:04d}"
        n_samples = int(duration_ms * sample_rate_hz / 1000)
        time_ms = np.linspace(0, duration_ms, n_samples)
        time_unix_ns = (time_ms * 1_000_000).astype(np.int64)
        
        # Generate base signals
        ip = 1.0 + 0.1 * np.sin(2 * np.pi * time_ms / 2000) + 0.05 * np.random.randn(n_samples)
        ne = 0.5 + 0.1 * np.sin(2 * np.pi * time_ms / 3000) + 0.03 * np.random.randn(n_samples)
        dwdt = 0.0 + 0.01 * np.random.randn(n_samples)
        prad = 0.3 + 0.05 * np.sin(2 * np.pi * time_ms / 4000) + 0.02 * np.random.randn(n_samples)
        h_alpha = 0.1 + 0.02 * np.random.randn(n_samples)
        
        # Determine if this shot has a disruption
        has_disruption = random.random() < disruption_probability
        
        if has_disruption:
            # Disruption occurs at random time in last 30% of shot
            disruption_time_ms = duration_ms * (0.7 + 0.3 * random.random())
            disruption_time_unix_ns = int(disruption_time_ms * 1_000_000)
            
            # Add disruption precursors
            disruption_idx = int(disruption_time_ms * sample_rate_hz / 1000)
            precursor_start = max(0, disruption_idx - int(1000 * sample_rate_hz / 1000))  # 1 second before
            
            # Increase radiation and decrease plasma current before disruption
            for i in range(precursor_start, min(disruption_idx, n_samples)):
                progress = (i - precursor_start) / (disruption_idx - precursor_start)
                prad[i] += 0.5 * progress
                ip[i] -= 0.2 * progress
                h_alpha[i] += 0.1 * progress
        else:
            disruption_time_unix_ns = None
        
        shot_df = pd.DataFrame({
            "shot_id": shot_id,
            "time_unix_ns": time_unix_ns,
            "ip": ip,
            "ne": ne,
            "dwdt": dwdt,
            "prad": prad,
            "h_alpha": h_alpha,
        })
        
        if disruption_time_unix_ns is not None:
            shot_df["disruption_time_unix_ns"] = disruption_time_unix_ns
            shot_df["time_to_disruption"] = (disruption_time_unix_ns - time_unix_ns) / 1_000_000
        else:
            shot_df["time_to_disruption"] = np.inf
        
        all_shots.append(shot_df)
    
    return pd.concat(all_shots, ignore_index=True)


def load_data(filepath: Path | str, format: str | None = None, **kwargs: Any) -> pd.DataFrame:
    """Load data from file, auto-detecting format if not specified.
    
    Args:
        filepath: Path to data file
        format: Format override ("csv", "hdf5", "netcdf", "synthetic")
        **kwargs: Additional arguments passed to specific loader
    """
    filepath = Path(filepath)
    
    if format is None:
        # Auto-detect format
        suffix = filepath.suffix.lower()
        if suffix == ".csv":
            format = "csv"
        elif suffix in [".h5", ".hdf5"]:
            format = "hdf5"
        elif suffix in [".nc", ".netcdf"]:
            format = "netcdf"
        else:
            raise ValueError(f"Could not auto-detect format for {filepath}")
    
    if format == "csv":
        return load_csv(filepath, **kwargs)
    elif format == "hdf5":
        return load_hdf5(filepath, **kwargs)
    elif format == "netcdf":
        return load_netcdf(filepath, **kwargs)
    elif format == "synthetic":
        return generate_synthetic_data(**kwargs)
    else:
        raise ValueError(f"Unknown format: {format}")
