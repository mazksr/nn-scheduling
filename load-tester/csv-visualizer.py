#!/usr/bin/env python3
# plot_k6_rps_latency_failed.py
#
# Usage:
#   python plot_k6_rps_latency_failed.py /path/to/k6.csv
#
# CSV columns expected (min):
#   metric_name,timestamp,metric_value,status
#
# Plots:
#   1) RPS over time (sum of http_reqs per second)
#   2) Latency over time (p95 and avg of http_req_duration per second)
#      + overlay (right Y-axis): failed HTTP requests per second (status >= 400)

import argparse
import math
from datetime import timezone

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt


def load_data(path: str) -> pd.DataFrame:
    df = pd.read_csv(path)
    df.columns = [c.strip().lower() for c in df.columns]

    # Required columns
    required = {"metric_name", "timestamp", "metric_value"}
    missing = required - set(df.columns)
    if missing:
        raise ValueError(f"CSV missing columns: {missing}")

    # Types
    df["metric_name"] = df["metric_name"].astype(str)
    df["timestamp"] = pd.to_numeric(df["timestamp"], errors="coerce").astype("Int64")
    df["metric_value"] = pd.to_numeric(df["metric_value"], errors="coerce")
    # Optional status column
    if "status" in df.columns:
        df["status"] = pd.to_numeric(df["status"], errors="coerce")

    df = df.dropna(subset=["timestamp", "metric_value"])
    # epoch seconds → naive UTC datetime for plotting
    df["dt"] = pd.to_datetime(df["timestamp"].astype(int), unit="s", utc=True)\
                 .dt.tz_convert(timezone.utc).dt.tz_localize(None)
    return df


def aggregate_rps(df: pd.DataFrame) -> pd.DataFrame:
    # RPS = sum of http_reqs per second
    rps = (
        df.loc[df["metric_name"] == "http_reqs", ["dt", "metric_value"]]
          .groupby("dt", as_index=False)["metric_value"].sum()
          .rename(columns={"metric_value": "rps"})
          .sort_values("dt")
    )
    return rps


def percentile(a: np.ndarray, q: float) -> float:
    if a.size == 0:
        return math.nan
    return float(np.percentile(a, q))


def aggregate_latency(df: pd.DataFrame) -> pd.DataFrame:
    # Per-second p95 & avg for http_req_duration (ms)
    sub = df.loc[df["metric_name"] == "http_req_duration", ["dt", "metric_value"]].copy()
    grouped = sub.groupby("dt")
    lat = pd.DataFrame({
        "dt": grouped.size().index,
        "p95_ms": grouped["metric_value"].apply(lambda s: percentile(s.to_numpy(), 95.0)).values,
        "avg_ms": grouped["metric_value"].mean().values,
        "count": grouped.size().values,
    }).sort_values("dt")
    return lat


def aggregate_failed_per_sec(df: pd.DataFrame) -> pd.DataFrame:
    # Prefer counting failures from http_reqs rows with status >= 400.
    if "status" not in df.columns:
        # No status info → empty failures series
        return pd.DataFrame(columns=["dt", "failed_per_s"])

    reqs = df.loc[df["metric_name"] == "http_reqs", ["dt", "metric_value", "status"]].copy()
    # In some exports, http_reqs rows have metric_value=1 per request; if not, we still sum metric_value for failed.
    reqs["is_failed"] = (reqs["status"] >= 400)
    failed = (
        reqs.loc[reqs["is_failed"]]
            .groupby("dt", as_index=False)["metric_value"]
            .sum()
            .rename(columns={"metric_value": "failed_per_s"})
            .sort_values("dt")
    )
    return failed


def plot_rps(rps: pd.DataFrame):
    plt.figure()
    plt.plot(rps["dt"], rps["rps"], marker="o", linewidth=1)
    plt.title("Requests Per Second (RPS)")
    plt.xlabel("Time (UTC)")
    plt.ylabel("RPS")
    plt.grid(True)
    plt.tight_layout()


def plot_latency_with_failures(lat: pd.DataFrame, fails: pd.DataFrame):
    # Merge failures into latency timeline (outer join to keep full time range)
    merged = pd.merge(lat, fails, on="dt", how="outer").sort_values("dt")
    merged["failed_per_s"] = merged["failed_per_s"].fillna(0)

    fig, ax1 = plt.subplots()
    ax1.plot(merged["dt"], merged["p95_ms"], label="p95", marker="o", linewidth=1)
    ax1.plot(merged["dt"], merged["avg_ms"], label="avg", marker="x", linewidth=1)
    ax1.set_title("Latency over Time (http_req_duration) + Failed Requests")
    ax1.set_xlabel("Time (UTC)")
    ax1.set_ylabel("Latency (ms)")
    ax1.grid(True)

    # Secondary Y-axis for failures per second
    ax2 = ax1.twinx()
    ax2.plot(merged["dt"], merged["failed_per_s"], label="failed/s", linestyle="--", linewidth=1)
    ax2.set_ylabel("Failed requests per second")

    # Combined legend
    lines1, labels1 = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax1.legend(lines1 + lines2, labels1 + labels2, loc="best")

    fig.tight_layout()


def main():
    ap = argparse.ArgumentParser(description="Plot k6 RPS and latency with failed requests overlay.")
    ap.add_argument("csv_path", help="Path to k6 CSV file")
    args = ap.parse_args()

    df = load_data(args.csv_path)
    rps = aggregate_rps(df)
    lat = aggregate_latency(df)
    fails = aggregate_failed_per_sec(df)

    if not rps.empty:
        plot_rps(rps)
    else:
        print("No http_reqs rows found; RPS plot will be empty.")

    if lat.empty:
        print("No http_req_duration rows found; latency plot will be empty.")
    else:
        plot_latency_with_failures(lat, fails)

    plt.show()


if __name__ == "__main__":
    main()

