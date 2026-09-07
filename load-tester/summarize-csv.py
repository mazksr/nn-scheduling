import pandas as pd
import argparse
import sys

def main():
    # Set up argument parsing to grab the path from args
    parser = argparse.ArgumentParser(description="Process k6 load test results.")
    parser.add_argument("filepath", help="Path to the k6 results CSV file")
    args = parser.parse_args()

    # 1. Load the messy k6 CSV
    try:
        df = pd.read_csv(args.filepath)
    except FileNotFoundError:
        print(f"Error: Could not find the file at '{args.filepath}'")
        sys.exit(1)

    # 2. Filter down to just the essential columns
    df_clean = df[['metric_name', 'metric_value', 'name']]

    # 4. Extract just the 'http_req_waiting' metric (the actual queue + inference time)
    df_latency = df_clean[df_clean['metric_name'] == 'http_req_waiting']

    # Check if we actually found any data
    if df_latency.empty:
        print("Warning: No data found for the specified endpoint and metric.")
        sys.exit(0)

    # 5. Generate the summary statistics for your thesis table
    summary = df_latency['metric_value'].describe(percentiles=[.50, .90, .95, .99])

    print("--- Latency Results for Least Connections ---")
    print(f"Average (Mean) Latency: {summary['mean']:.2f} ms")
    print(f"Max Latency:            {summary['max']:.2f} ms")
    print(f"P95 Latency:            {summary['95%']:.2f} ms")

if __name__ == "__main__":
    main()
