#!/usr/bin/env python3
import os
import sys
import glob
import pandas as pd
import numpy as np
import matplotlib
matplotlib.use('Agg')  # Paksa mode headless agar tidak crash/hang tanpa GUI
import matplotlib.pyplot as plt

# Konfigurasi gaya visualisasi standar publikasi akademik
plt.rcParams['font.sans-serif'] = 'DejaVu Sans'
plt.rcParams['axes.edgecolor'] = '#333333'
plt.rcParams['axes.linewidth'] = 0.8

# Pemetaan fleksibel (mendukung nama file dengan atau tanpa prefix 'infer-')
STATEFUL_ALGORITHMS = [
    {
        'keys': ['infer-leastconn', 'leastconn', 'lc'],
        'label': 'Least Connections (LC)',
        'color': '#e03131'
    },
    {
        'keys': ['infer-p2c', 'p2c'],
        'label': 'Power of Two Choices (P2C)',
        'color': '#f08c00'
    },
    {
        'keys': ['infer-hybrid', 'hybrid'],
        'label': 'Hybrid Payload-Aware',
        'color': '#1971c2'
    }
]

def find_csv_file(data_dir, candidate_keys):
    """Mencari berkas CSV yang cocok di dalam direktori."""
    for key in candidate_keys:
        exact = os.path.join(data_dir, f"k6_{key}.csv")
        if os.path.exists(exact):
            return exact
        # Cek pola wildcard jika nama berkas sedikit berbeda
        matches = glob.glob(os.path.join(data_dir, f"*{key}*.csv"))
        if matches:
            return matches[0]
    return None

def load_timeseries_stateful(data_dir):
    timeseries_data = []

    for meta in STATEFUL_ALGORITHMS:
        csv_path = find_csv_file(data_dir, meta['keys'])
        if not csv_path:
            print(f"[!] Peringatan: CSV untuk {meta['label']} tidak ditemukan di {data_dir}", flush=True)
            continue

        print(f"==> Memuat berkas: {csv_path}", flush=True)
        df = pd.read_csv(csv_path, low_memory=False)

        df_dur = df[df['metric_name'] == 'http_req_duration'].copy()
        if df_dur.empty:
            print(f"[!] Peringatan: Tidak ada metrik http_req_duration di {csv_path}", flush=True)
            continue

        df_dur['metric_value'] = pd.to_numeric(df_dur['metric_value'], errors='coerce')
        df_dur['timestamp'] = pd.to_numeric(df_dur['timestamp'], errors='coerce')

        min_ts = df_dur['timestamp'].min()
        df_dur['rel_sec'] = df_dur['timestamp'] - min_ts

        ts_grouped = df_dur.groupby('rel_sec')['metric_value'].agg(mean='mean').reset_index()
        ts_grouped['mean_smooth'] = ts_grouped['mean'].rolling(window=3, min_periods=1).mean()

        timeseries_data.append((meta, ts_grouped))

    return timeseries_data

def plot_zoomed_timeseries(timeseries_data, out_dir):
    fig, ax = plt.subplots(figsize=(12, 5.5), dpi=300)

    for meta, ts_df in timeseries_data:
        is_hybrid = 'Hybrid' in meta['label']
        lw = 2.5 if is_hybrid else 1.6
        alpha = 1.0 if is_hybrid else 0.85
        zorder = 5 if is_hybrid else 3
        ax.plot(
            ts_df['rel_sec'], ts_df['mean_smooth'],
            label=meta['label'], color=meta['color'],
            linewidth=lw, alpha=alpha, zorder=zorder
        )

    y_limit = 750
    ax.set_ylim(0, y_limit)
    ax.set_xlim(0, 185)

    stages = [
        (30, 'Stage 1: Warm-up\n(->10 RPS)'),
        (90, 'Stage 2: Sustained\n(->24 RPS)'),
        (120, 'Stage 3: Peak Burst\n(->60 RPS)'),
        (150, 'Stage 4: High Load\n(60 RPS)'),
        (180, 'Stage 5: Cooldown\n(->10 RPS)')
    ]
    prev_t = 0
    for t_bound, stage_label in stages:
        if t_bound < 180:
            ax.axvline(x=t_bound, color='#555555', linestyle=':', linewidth=1.0)
        mid_t = (prev_t + t_bound) / 2
        ax.text(
            mid_t, y_limit * 0.92, stage_label, ha='center', va='top',
            fontsize=8, bbox=dict(boxstyle='round,pad=0.2', facecolor='#f8f9fa', edgecolor='#dee2e6', alpha=0.85)
        )
        prev_t = t_bound

    ax.set_xlabel('Durasi Pengujian (Detik)', fontsize=11, fontweight='bold')
    ax.set_ylabel('Rata-Rata Waktu Respons / Latensi (ms)', fontsize=11, fontweight='bold')
    ax.set_title(
        'Tren Latensi Rata-Rata Algoritma Stateful (LC, P2C, & Hybrid Payload-Aware) [Skala 0–750 ms]',
        fontsize=12.5, fontweight='bold', pad=15
    )
    ax.legend(loc='upper left', frameon=True, facecolor='white', edgecolor='#cccccc', fontsize=9.5)
    ax.grid(True, linestyle='--', alpha=0.4)

    plt.tight_layout()
    out_path = os.path.join(out_dir, 'grafik_2b_tren_waktu_latensi_zoom.png')
    plt.savefig(out_path, dpi=300)
    plt.close()
    print(f"==> Selesai! Grafik Zoomed-In disimpan ke: {out_path}", flush=True)

if __name__ == '__main__':
    print("==> Memulai skrip plot_zoomed_latency.py...", flush=True)
    target_dir = sys.argv[1] if len(sys.argv) > 1 else './k6_csv'
    print(f"==> Direktori target: {target_dir}", flush=True)

    if not os.path.exists(target_dir):
        print(f"[!] Error: Direktori {target_dir} tidak ditemukan!", flush=True)
        sys.exit(1)

    print(f"==> Isi direktori: {os.listdir(target_dir)}", flush=True)
    data = load_timeseries_stateful(target_dir)

    if not data:
        print("[!] Gagal: Tidak ada data CSV yang berhasil dimuat.", flush=True)
        sys.exit(1)

    plot_zoomed_timeseries(data, target_dir)
