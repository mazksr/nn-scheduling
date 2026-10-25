#!/usr/bin/env python3
import os
import sys
import glob
import re
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

# Konfigurasi gaya visualisasi standar publikasi akademik
plt.rcParams['font.sans-serif'] = 'DejaVu Sans'
plt.rcParams['axes.edgecolor'] = '#333333'
plt.rcParams['axes.linewidth'] = 0.8

# Pemetaan nama endpoint ke label akademis dan palet warna
ALGORITHMS = {
    'infer-rr': {'label': 'Round Robin (RR)', 'color': '#6c757d'},
    'infer-leastconn': {'label': 'Least Connections (LC)', 'color': '#e03131'},
    'infer-p2c': {'label': 'Power of Two Choices (P2C)', 'color': '#f08c00'},
    'infer-hybrid': {'label': 'Hybrid Payload-Aware', 'color': '#1971c2'}
}

def load_k6_csv(filepath):
    """Membaca berkas CSV mentah dari k6 dengan tipe data yang dioptimalkan."""
    print(f"==> Memuat berkas: {filepath}")
    df = pd.read_csv(filepath, low_memory=False)
    return df

def extract_worker_id(extra_tags):
    """Mengekstrak ID worker dari kolom extra_tags (contoh: 'worker=node-1')."""
    if pd.isna(extra_tags):
        return 'unknown'
    match = re.search(r'worker=([^,\s]+)', str(extra_tags))
    return match.group(1) if match else 'unknown'

def process_all_data(data_dir):
    summary_rows = []
    timeseries_data = {}
    worker_dist_data = {}

    for alg_key, meta in ALGORITHMS.items():
        csv_path = os.path.join(data_dir, f"k6_{alg_key}.csv")
        if not os.path.exists(csv_path):
            print(f"[!] Peringatan: Berkas {csv_path} tidak ditemukan, melewati...")
            continue

        df = load_k6_csv(csv_path)

        # 1. Filter metrik durasi HTTP (http_req_duration) dalam satuan milidetik (ms)
        df_dur = df[df['metric_name'] == 'http_req_duration'].copy()
        df_dur['metric_value'] = pd.to_numeric(df_dur['metric_value'], errors='coerce')
        df_dur['timestamp'] = pd.to_numeric(df_dur['timestamp'], errors='coerce')

        # Filter kegagalan permintaan (http_req_failed)
        df_fail = df[df['metric_name'] == 'http_req_failed'].copy()
        total_reqs = len(df_dur)
        failed_reqs = int(df_fail['metric_value'].sum()) if not df_fail.empty else 0
        error_rate = (failed_reqs / total_reqs * 100) if total_reqs > 0 else 0.0

        # Hitung statistik agregat QoS
        mean_val = df_dur['metric_value'].mean()
        p50_val = df_dur['metric_value'].median()
        p95_val = np.percentile(df_dur['metric_value'].dropna(), 95)
        p99_val = np.percentile(df_dur['metric_value'].dropna(), 99)
        max_val = df_dur['metric_value'].max()

        summary_rows.append({
            'Algoritma': meta['label'],
            'Total Permintaan': total_reqs,
            'Kegagalan (%)': round(error_rate, 2),
            'Mean (ms)': round(mean_val, 2),
            'Median / P50 (ms)': round(p50_val, 2),
            'P95 (ms)': round(p95_val, 2),
            'P99 (ms)': round(p99_val, 2),
            'Max Latency (ms)': round(max_val, 2)
        })

        # 2. Normalisasi deret waktu (Time-Series) ke detik relatif (0 - 180s)
        min_ts = df_dur['timestamp'].min()
        df_dur['rel_sec'] = df_dur['timestamp'] - min_ts
        # Agregasi rata-rata dan P95 per detik untuk grafik tren
        ts_grouped = df_dur.groupby('rel_sec')['metric_value'].agg(
            mean='mean',
            p95=lambda x: np.percentile(x, 95),
            max='max'
        ).reset_index()
        # Haluskan garis dengan rolling window 3 detik
        ts_grouped['mean_smooth'] = ts_grouped['mean'].rolling(window=3, min_periods=1).mean()
        ts_grouped['p95_smooth'] = ts_grouped['p95'].rolling(window=3, min_periods=1).mean()
        timeseries_data[alg_key] = ts_grouped

        # 3. Ekstraksi distribusi per Worker Node (metrik: worker_requests)
        df_workers = df[df['metric_name'] == 'worker_requests'].copy()
        if not df_workers.empty:
            if 'worker' in df_workers.columns and df_workers['worker'].notna().any():
                df_workers['worker_id'] = df_workers['worker']
            else:
                df_workers['worker_id'] = df_workers['extra_tags'].apply(extract_worker_id)
            worker_counts = df_workers.groupby('worker_id')['metric_value'].sum().to_dict()
            worker_dist_data[meta['label']] = worker_counts

    df_summary = pd.DataFrame(summary_rows)
    return df_summary, timeseries_data, worker_dist_data

def plot_qos_comparison(df_summary, out_dir):
    """Membuat diagram batang komparasi Mean, P50, P95, dan Max Latency."""
    metrics = ['Median / P50 (ms)', 'Mean (ms)', 'P95 (ms)', 'Max Latency (ms)']
    x = np.arange(len(metrics))
    width = 0.2

    fig, ax = plt.subplots(figsize=(11, 6), dpi=300)

    alg_keys = list(ALGORITHMS.keys())
    for i, (_, row) in enumerate(df_summary.iterrows()):
        alg_label = row['Algoritma']
        color = next((v['color'] for v in ALGORITHMS.values() if v['label'] == alg_label), '#333333')
        values = [row[m] for m in metrics]
        offset = (i - len(df_summary) / 2 + 0.5) * width
        bars = ax.bar(x + offset, values, width, label=alg_label, color=color, edgecolor='black', linewidth=0.5)

        # Tambahkan label angka di atas setiap batang
        for bar in bars:
            height = bar.get_height()
            ax.annotate(f'{height:.1f}',
                        xy=(bar.get_x() + bar.get_width() / 2, height),
                        xytext=(0, 4),
                        textcoords="offset points",
                        ha='center', va='bottom', fontsize=7.5, rotation=0)

    ax.set_ylabel('Waktu Respons / Latensi (ms)', fontsize=11, fontweight='bold')
    ax.set_title('Komparasi Metrik Latensi (QoS) Antar Algoritma Load Balancing', fontsize=13, fontweight='bold', pad=15)
    ax.set_xticks(x)
    ax.set_xticklabels(metrics, fontsize=10, fontweight='bold')
    ax.legend(frameon=True, facecolor='white', edgecolor='#cccccc', fontsize=9.5)
    ax.grid(axis='y', linestyle='--', alpha=0.5)
    ax.set_ylim(0, df_summary['Max Latency (ms)'].max() * 1.15)

    plt.tight_layout()
    out_path = os.path.join(out_dir, 'grafik_1_komparasi_qos.png')
    plt.savefig(out_path)
    plt.close()
    print(f"==> Grafik 1 disimpan ke: {out_path}")

def plot_timeseries_latency(timeseries_data, out_dir):
    """Membuat grafik deret waktu latensi selama 180 detik skenario pengujian."""
    fig, ax = plt.subplots(figsize=(12, 6), dpi=300)

    for alg_key, ts_df in timeseries_data.items():
        meta = ALGORITHMS[alg_key]
        lw = 2.4 if alg_key == 'infer-hybrid' else 1.5
        alpha = 1.0 if alg_key == 'infer-hybrid' else 0.8
        ax.plot(ts_df['rel_sec'], ts_df['mean_smooth'], label=meta['label'],
                color=meta['color'], linewidth=lw, alpha=alpha)

    # Garis vertikal penanda tahapan beban kerja (stages k6)
    stages = [
        (30, 'Stage 1: Warm-up\n(->10 RPS)'),
        (90, 'Stage 2: Sustained\n(->24 RPS)'),
        (120, 'Stage 3: Peak Burst\n(->60 RPS)'),
        (150, 'Stage 4: High Load\n(60 RPS)'),
        (180, 'Stage 5: Cooldown\n(->10 RPS)')
    ]
    prev_t = 0
    y_max = ax.get_ylim()[1]
    for t_bound, stage_label in stages:
        if t_bound < 180:
            ax.axvline(x=t_bound, color='#555555', linestyle=':', linewidth=1.0)
        mid_t = (prev_t + t_bound) / 2
        ax.text(mid_t, y_max * 0.92, stage_label, ha='center', va='top',
                fontsize=8, bbox=dict(boxstyle='round,pad=0.2', facecolor='#f8f9fa', edgecolor='#dee2e6', alpha=0.8))
        prev_t = t_bound

    ax.set_xlabel('Durasi Pengujian (Detik)', fontsize=11, fontweight='bold')
    ax.set_ylabel('Rata-Rata Waktu Respons / Latensi (ms)', fontsize=11, fontweight='bold')
    ax.set_title('Tren Latensi Rata-Rata Terhadap Tahapan Injeksi Beban (ramping-arrival-rate)', fontsize=13, fontweight='bold', pad=15)
    ax.set_xlim(0, 185)
    ax.legend(loc='upper left', frameon=True, facecolor='white', edgecolor='#cccccc', fontsize=9.5)
    ax.grid(True, linestyle='--', alpha=0.4)

    plt.tight_layout()
    out_path = os.path.join(out_dir, 'grafik_2_tren_waktu_latensi.png')
    plt.savefig(out_path)
    plt.close()
    print(f"==> Grafik 2 disimpan ke: {out_path}")

def plot_worker_distribution(worker_dist_data, out_dir):
    """Membuat grafik distribusi permintaan di ketiga Worker Node."""
    if not worker_dist_data:
        print("[!] Tidak ada data metrik worker_requests untuk divisualisasikan.")
        return

    df_w = pd.DataFrame(worker_dist_data).T.fillna(0)
    ax = df_w.plot(kind='bar', figsize=(10, 5.5), width=0.65, edgecolor='black', linewidth=0.6)
    fig = ax.get_figure()
    fig.set_dpi(300)

    ax.set_ylabel('Jumlah Permintaan (Requests)', fontsize=11, fontweight='bold')
    ax.set_xlabel('Algoritma Load Balancing', fontsize=11, fontweight='bold')
    ax.set_title('Distribusi Permintaan HTTP pada Klaster Worker Node', fontsize=13, fontweight='bold', pad=15)
    plt.xticks(rotation=15, ha='right', fontsize=9.5, fontweight='bold')
    ax.legend(title='ID Worker Node', frameon=True, facecolor='white', edgecolor='#cccccc')
    ax.grid(axis='y', linestyle='--', alpha=0.5)

    for container in ax.containers:
        ax.bar_label(container, fmt='%.0f', fontsize=8, padding=3)

    plt.tight_layout()
    out_path = os.path.join(out_dir, 'grafik_3_distribusi_worker.png')
    plt.savefig(out_path)
    plt.close()
    print(f"==> Grafik 3 disimpan ke: {out_path}")

def main():
    # Ambil direktori terbaru di ./k6_csv/ jika argumen tidak diberikan
    if len(sys.argv) > 1:
        target_dir = sys.argv[1]
    else:
        subdirs = sorted(glob.glob('./k6_csv/*'))
        if not subdirs:
            print("Penggunaan: python3 analyze_results.py <path_ke_folder_csv>")
            sys.exit(1)
        target_dir = subdirs[-1]

    print(f"==> Memproses direktori hasil pengujian: {target_dir}\n")
    df_summary, timeseries_data, worker_dist_data = process_all_data(target_dir)

    if df_summary.empty:
        print("[!] Tidak ada data valid yang dapat diproses.")
        sys.exit(1)

    # Cetak tabel ke terminal & simpan sebagai CSV untuk Bab 3
    print("\n================ TABEL RINGKASAN METRIK QoS (BAB III) ================")
    print(df_summary.to_string(index=False))
    print("======================================================================\n")

    summary_csv_path = os.path.join(target_dir, 'tabel_ringkasan_qos_bab3.csv')
    df_summary.to_csv(summary_csv_path, index=False)
    print(f"==> Tabel ringkasan disimpan ke: {summary_csv_path}")

    # Hasilkan grafik visualisasi
    plot_qos_comparison(df_summary, target_dir)
    plot_timeseries_latency(timeseries_data, target_dir)
    plot_worker_distribution(worker_dist_data, target_dir)

if __name__ == '__main__':
    main()
