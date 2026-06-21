import torch
from fastapi import FastAPI, File, UploadFile
from transformers import ViTImageProcessor, ViTForImageClassification
from PIL import Image
import io
import uvicorn
import argparse
from contextlib import asynccontextmanager
import asyncio
from concurrent.futures import ThreadPoolExecutor

executor = ThreadPoolExecutor(max_workers=4)

# =====================================================
# CLI arguments
# =====================================================
parser = argparse.ArgumentParser(description="ViT Inference Server")
parser.add_argument("--port", type=int, default=8000, help="Port to listen on")
args = parser.parse_args()

# =====================================================
# Load model and processor
# =====================================================
MODEL_DIR = "./vit-base-patch16-224"
DEVICE = torch.device("cuda" if torch.cuda.is_available() else "cpu")
print(f"Loading model from {MODEL_DIR} on {DEVICE}...")
processor = ViTImageProcessor.from_pretrained(MODEL_DIR)
model = ViTForImageClassification.from_pretrained(MODEL_DIR)
model.eval().to(DEVICE)

@asynccontextmanager
async def lifespan(app: FastAPI):
    asyncio.get_event_loop().set_default_executor(executor)
    yield

# =====================================================
# FastAPI app
# =====================================================
app = FastAPI(lifespan=lifespan, title="ViT Image Classification API")

@app.get("/health")
async def health():
    """
    Health check endpoint for the scheduler.
    """
    return {"status": "ok"}

def run_inference(image_bytes: bytes):
    img = Image.open(io.BytesIO(image_bytes)).convert("RGB")
    inputs = processor(images=img, return_tensors="pt").to(DEVICE)
    with torch.inference_mode():
        outputs = model(**inputs)
        logits = outputs.logits
        pred_idx = int(logits.argmax(-1))
        label = model.config.id2label[pred_idx]
    return {"class": label, "index": pred_idx}

@app.post("/infer")
async def infer(file: UploadFile = File(...)):
    """
    Accepts an uploaded image, runs inference,
    and returns the predicted ImageNet class in JSON.
    """
    image_bytes = await file.read()
    result = await asyncio.to_thread(run_inference, image_bytes)
    return result

# =====================================================
# Entry point
# =====================================================
if __name__ == "__main__":
    print(f"Starting server on port {args.port} ...")
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=args.port,
        workers=1,
        reload=False,
    )

