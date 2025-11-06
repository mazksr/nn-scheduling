#!/usr/bin/env python3
# plot_k6_rps_latency_failed_reltime.py
#
# Usage:
#   python plot_k6_rps_latency_failed_reltime.py /path/to/k6.csv
#
# CSV columns expected (min):
#   metric_name,timestamp,metric_value[,status]
#
# Plots:
#   1) RPS over time (sum of http_reqs per second), x-axis 0m0s..XmYs
#   2) Latency over time (p95 & avg) with failed/s overlay (status >= 400), x-axis 0m0s..XmYs

import argparse
import math
from datetime import timezone

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from matplotlib.ticker import FuncFormatter


def mmss_formatter():
    def _fmt(x, _pos):
        # x is seconds (float)
        x = max(0.0, float(x))
        m = int(x // 60)
        s = int(round(x % 60))
        return f"{m}m{s:02d}s"
    return FuncFormatter(_fmt)


def load_data(path: str) -> pd.DataFrame:
    df = pd.read_csv(path)
    df.columns = [c.strip().lower() for c in df.columns]

    required = {"metric_name", "timestamp", "metric_value"}
    missing = required - set(df.columns)
    if missing:
        raise ValueError(f"CSV missing columns: {missing}")

    df["metric_name"] = df["metric_name"].astype(str)
    df["timestamp"] = pd.to_numeric(df["timestamp"], errors="coerce").astype("Int64")
    df["metric_value"] = pd.to_numeric(df["metric_value"], errors="coerce")
    if "status" in df.columns:
        df["status"] = pd.to_numeric(df["status"], errors="coerce")

    df = df.dropna(subset=["timestamp", "metric_value"])

    # epoch seconds → naive UTC datetime
    df["dt"] = pd.to_datetime(df["timestamp"].astype(int), unit="s", utc=True)\
                 .dt.tz_convert(timezone.utc).dt.tz_localize(None)

    # Compute relative seconds from test start
    t0 = df["dt"].min()
    df["t_rel"] = (df["dt"] - t0).dt.total_seconds()

    return df


def test_duration_seconds(df: pd.DataFrame) -> float:
    # Span of observed timestamps
    t_min = df["t_rel"].min() if not df.empty else 0.0
    t_max = df["t_rel"].max() if not df.empty else 0.0
    dur = float(max(0.0, t_max - t_min))
    # Avoid zero-width axis; show at least 1s
    return dur if dur > 0 else 1.0


def aggregate_rps(df: pd.DataFrame) -> pd.DataFrame:
    rps = (
        df.loc[df["metric_name"] == "http_reqs", ["t_rel", "metric_value"]]
          .groupby("t_rel", as_index=False)["metric_value"].sum()
          .rename(columns={"metric_value": "rps"})
          .sort_values("t_rel")
    )
    return rps


def percentile(a: np.ndarray, q: float) -> float:
    if a.size == 0:
        return math.nan
    return float(np.percentile(a, q))


def aggregate_latency(df: pd.DataFrame) -> pd.DataFrame:
    sub = df.loc[df["metric_name"] == "http_req_duration", ["t_rel", "metric_value"]].copy()
    grouped = sub.groupby("t_rel")
    lat = pd.DataFrame({
        "t_rel": grouped.size().index,
        "p95_ms": grouped["metric_value"].apply(lambda s: percentile(s.to_numpy(), 95.0)).values,
        "avg_ms": grouped["metric_value"].mean().values,
        "count": grouped.size().values,
    }).sort_values("t_rel")
    return lat


def aggregate_failed_per_sec(df: pd.DataFrame) -> pd.DataFrame:
    if "status" not in df.columns:
        return pd.DataFrame(columns=["t_rel", "failed_per_s"])

    reqs = df.loc[df["metric_name"] == "http_reqs", ["t_rel", "metric_value", "status"]].copy()
    reqs["is_failed"] = (reqs["status"] >= 400)
    failed = (
        reqs.loc[reqs["is_failed"]]
            .groupby("t_rel", as_index=False)["metric_value"]
            .sum()
            .rename(columns={"metric_value": "failed_per_s"})
            .sort_values("t_rel")
    )
    return failed


def plot_rps(rps: pd.DataFrame, duration_s: float):
    plt.figure()
    plt.plot(rps["t_rel"], rps["rps"], marker="o", linewidth=1)
    plt.title("Requests Per Second (RPS)")
    plt.xlabel("Time since start")
    plt.ylabel("RPS")
    ax = plt.gca()
    ax.xaxis.set_major_formatter(mmss_formatter())
    ax.set_xlim(0, duration_s)
    plt.grid(True)
    plt.tight_layout()


def plot_latency_with_failures(lat: pd.DataFrame, fails: pd.DataFrame, duration_s: float):
    merged = pd.merge(lat, fails, on="t_rel", how="outer").sort_values("t_rel")
    merged["failed_per_s"] = merged["failed_per_s"].fillna(0)

    fig, ax1 = plt.subplots()
    ax1.plot(merged["t_rel"], merged["p95_ms"], label="p95", marker="o", linewidth=1)
    ax1.plot(merged["t_rel"], merged["avg_ms"], label="avg", marker="x", linewidth=1)
    ax1.set_title("Latency (http_req_duration) + Failed Requests")
    ax1.set_xlabel("Time since start")
    ax1.set_ylabel("Latency (ms)")
    ax1.grid(True)
    ax1.xaxis.set_major_formatter(mmss_formatter())
    ax1.set_xlim(0, duration_s)

    ax2 = ax1.twinx()
    ax2.plot(merged["t_rel"], merged["failed_per_s"], label="failed/s", linestyle="--", linewidth=1)
    ax2.set_ylabel("Failed requests per second")

    lines1, labels1 = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax1.legend(lines1 + lines2, labels1 + labels2, loc="best")

    fig.tight_layout()


def main():
    ap = argparse.ArgumentParser(description="Plot k6 RPS and latency (relative time 0m0s..XmYs) with failed requests overlay.")
    ap.add_argument("csv_path", help="Path to k6 CSV file")
    args = ap.parse_args()

    df = load_data(args.csv_path)
    duration_s = test_duration_seconds(df)

    rps = aggregate_rps(df)
    lat = aggregate_latency(df)
    fails = aggregate_failed_per_sec(df)

    if not rps.empty:
        plot_rps(rps, duration_s)
    else:
        print("No http_reqs rows found; RPS plot will be empty.")

    if lat.empty:
        print("No http_req_duration rows found; latency plot will be empty.")
    else:
        plot_latency_with_failures(lat, fails, duration_s)

    # Show overall duration in console
    m = int(duration_s // 60)
    s = int(round(duration_s % 60))
    print(f"Test duration: {m}m{s:02d}s")

    plt.show()


if __name__ == "__main__":
    main()
