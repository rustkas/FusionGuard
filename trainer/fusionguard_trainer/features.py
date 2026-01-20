from __future__ import annotations

import numpy as np
import pandas as pd
from scipy import signal, stats


def compute_window_features(
    df: pd.DataFrame,
    channels: list[str],
    windows_ms: list[int],
    sample_rate_hz: int = 1000,
) -> pd.DataFrame:
    """Compute sliding window features for telemetry channels.
    
    For each channel and window size, computes:
    - mean, std, min, max
    - slope (linear trend)
    - last value
    - delta (change from previous point)
    - z-score (deviation from baseline)
    """
    result = df.copy()
    
    for channel in channels:
        if channel not in df.columns:
            continue
        
        values = df[channel].values
        
        for window_ms in windows_ms:
            window_samples = int(window_ms * sample_rate_hz / 1000)
            
            # Rolling statistics
            rolling = df[channel].rolling(window=window_samples, min_periods=1)
            
            result[f"{channel}_mean_w{window_ms}"] = rolling.mean()
            result[f"{channel}_std_w{window_ms}"] = rolling.std().fillna(0)
            result[f"{channel}_min_w{window_ms}"] = rolling.min()
            result[f"{channel}_max_w{window_ms}"] = rolling.max()
            
            # Slope (linear trend over window)
            slopes = []
            for i in range(len(values)):
                start_idx = max(0, i - window_samples + 1)
                window_vals = values[start_idx : i + 1]
                if len(window_vals) >= 2:
                    x = np.arange(len(window_vals))
                    slope, _ = np.polyfit(x, window_vals, 1)
                    slopes.append(slope * sample_rate_hz / 1000)  # Convert to per-ms
                else:
                    slopes.append(0.0)
            result[f"{channel}_slope_w{window_ms}"] = slopes
            
            # Last value
            result[f"{channel}_last_w{window_ms}"] = rolling.apply(lambda x: x.iloc[-1] if len(x) > 0 else np.nan)
            
            # Delta (change from previous point)
            result[f"{channel}_delta_w{window_ms}"] = df[channel].diff().fillna(0)
            
            # Z-score (deviation from baseline - using EMA as baseline)
            ema_alpha = 0.01
            ema = df[channel].ewm(alpha=ema_alpha, adjust=False).mean()
            std_ema = df[channel].ewm(alpha=ema_alpha, adjust=False).std().fillna(1.0)
            result[f"{channel}_zscore_w{window_ms}"] = (df[channel] - ema) / (std_ema + 1e-9)
    
    return result


def compute_spectral_features(
    df: pd.DataFrame,
    channels: list[str],
    sample_rate_hz: int = 1000,
) -> pd.DataFrame:
    """Compute spectral features (energy, dominant frequency) for channels."""
    result = df.copy()
    
    for channel in channels:
        if channel not in df.columns:
            continue
        
        values = df[channel].values
        
        # Energy in different frequency bands
        # Use a sliding window for spectral analysis
        window_size = min(1000, len(values) // 10)  # 1 second or 10% of data
        
        energies = []
        dominant_freqs = []
        
        for i in range(len(values)):
            start_idx = max(0, i - window_size + 1)
            window_vals = values[start_idx : i + 1]
            
            if len(window_vals) >= 32:  # Minimum for FFT
                # Compute FFT
                fft_vals = np.fft.rfft(window_vals)
                freqs = np.fft.rfftfreq(len(window_vals), 1.0 / sample_rate_hz)
                power = np.abs(fft_vals) ** 2
                
                # Total energy
                total_energy = np.sum(power)
                energies.append(total_energy)
                
                # Dominant frequency
                if len(power) > 0:
                    dominant_idx = np.argmax(power[1:]) + 1  # Skip DC component
                    dominant_freqs.append(freqs[dominant_idx])
                else:
                    dominant_freqs.append(0.0)
            else:
                energies.append(0.0)
                dominant_freqs.append(0.0)
        
        result[f"{channel}_energy"] = energies
        result[f"{channel}_dominant_freq"] = dominant_freqs
    
    return result


def compute_anomaly_features(
    df: pd.DataFrame,
    channels: list[str],
) -> pd.DataFrame:
    """Compute anomaly detection features (isolation, deviation from normal)."""
    result = df.copy()
    
    for channel in channels:
        if channel not in df.columns:
            continue
        
        values = df[channel].values
        
        # Percentile-based anomaly score
        # Compare current value to distribution in recent window
        window_size = 1000  # 1 second at 1kHz
        
        anomaly_scores = []
        for i in range(len(values)):
            start_idx = max(0, i - window_size + 1)
            window_vals = values[start_idx : i + 1]
            
            if len(window_vals) >= 10:
                # Compute how far current value is from median
                median = np.median(window_vals)
                mad = np.median(np.abs(window_vals - median))  # Median Absolute Deviation
                if mad > 0:
                    score = abs(values[i] - median) / (mad + 1e-9)
                else:
                    score = 0.0
                anomaly_scores.append(score)
            else:
                anomaly_scores.append(0.0)
        
        result[f"{channel}_anomaly"] = anomaly_scores
    
    # Global missing ratio
    if "time_unix_ns" in df.columns:
        # Compute missing ratio per time point (simplified - assumes all channels should be present)
        result["missing_ratio"] = 0.0  # Will be computed based on actual missing data
    
    return result


def compute_all_features(
    df: pd.DataFrame,
    channels: list[str] | None = None,
    windows_ms: list[int] | None = None,
    sample_rate_hz: int = 1000,
) -> pd.DataFrame:
    """Compute all features for the dataset.
    
    Args:
        df: Input dataframe with telemetry data
        channels: List of channel names (auto-detect if None)
        windows_ms: List of window sizes in milliseconds
        sample_rate_hz: Sampling rate in Hz
    """
    if channels is None:
        # Auto-detect channels (exclude metadata columns)
        exclude = {"shot_id", "time_unix_ns", "time_to_disruption", "disruption_time_unix_ns", "time_ms", "time"}
        channels = [col for col in df.columns if col not in exclude]
    
    if windows_ms is None:
        windows_ms = [50, 200]
    
    # Compute window-based features
    result = compute_window_features(df, channels, windows_ms, sample_rate_hz)
    
    # Compute spectral features (optional, can be slow)
    # result = compute_spectral_features(result, channels, sample_rate_hz)
    
    # Compute anomaly features
    result = compute_anomaly_features(result, channels)
    
    return result
