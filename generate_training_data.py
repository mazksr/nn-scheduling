import os
import time
import csv
import requests
from PIL import Image
import argparse

def generate_training_data(image_dir, output_csv, server_url):
    if not os.path.isdir(image_dir):
        print(f"Error: Directory not found: {image_dir}")
        return

    print(f"Scanning '{image_dir}' for images...")
    print(f"Sending requests to: {server_url}")
    print(f"Output will be saved to: {output_csv}\n")

    with open(output_csv, mode='w', newline='', encoding='utf-8') as csv_file:
        fieldnames = [
            'filename', 
            'file_size_bytes', 
            'image_width', 
            'image_height', 
            'total_pixels', 
            'processing_time_ms'
        ]
        writer = csv.DictWriter(csv_file, fieldnames=fieldnames)
        writer.writeheader()

        valid_extensions = {'.png', '.jpg', '.jpeg', '.bmp', '.webp'}
        processed_count = 0

        for filename in os.listdir(image_dir):
            ext = os.path.splitext(filename)[1].lower()
            if ext not in valid_extensions:
                continue
            
            filepath = os.path.join(image_dir, filename)
            try:
                # 1. Feature: File Size
                file_size_bytes = os.path.getsize(filepath)

                # 2. Features: Image Dimensions
                with Image.open(filepath) as img:
                    image_width, image_height = img.size
                
                # 3. Feature: Total Pixels
                total_pixels = image_width * image_height

                # 4. Target: Processing Time (measure request round-trip)
                with open(filepath, 'rb') as f:
                    mime_type = 'image/jpeg' if ext in {'.jpg', '.jpeg'} else f'image/{ext[1:]}'
                    files = {'file': (filename, f, mime_type)}
                    
                    start_time = time.perf_counter()
                    response = requests.post(server_url, files=files)
                    response.raise_for_status()
                    end_time = time.perf_counter()

                processing_time_ms = (end_time - start_time) * 1000

                # 5. Record to CSV
                writer.writerow({
                    'filename': filename,
                    'file_size_bytes': file_size_bytes,
                    'image_width': image_width,
                    'image_height': image_height,
                    'total_pixels': total_pixels,
                    'processing_time_ms': round(processing_time_ms, 2)
                })
                
                print(f"[+] Processed {filename} | Size: {file_size_bytes}B | Pixels: {total_pixels} | Time: {processing_time_ms:.2f}ms")
                processed_count += 1

            except Exception as e:
                print(f"[-] Error processing {filename}: {e}")

    print(f"\nDone! Successfully processed {processed_count} images.")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Generate ML training data from image inference times.")
    parser.add_argument("--img-dir", type=str, required=True, help="Path to the directory containing images")
    parser.add_argument("--out", type=str, default="training_data.csv", help="Output CSV file path")
    parser.add_argument("--url", type=str, default="http://localhost:8080/infer-rr", help="The inference endpoint URL to send images to")
    
    args = parser.parse_args()
    generate_training_data(args.img_dir, args.out, args.url)
