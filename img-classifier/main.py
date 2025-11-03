import torch
from fastapi import FastAPI, File, UploadFile
from transformers import ViTImageProcessor, ViTForImageClassification
from PIL import Image
import io, uvicorn, argparse

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

# =====================================================
# FastAPI app
# =====================================================
app = FastAPI(title="ViT Image Classification API")

@app.post("/infer")
async def infer(file: UploadFile = File(...)):
    """
    Accepts an uploaded image, runs inference,
    and returns the predicted ImageNet class in JSON.
    """
    image_bytes = await file.read()
    image = Image.open(io.BytesIO(image_bytes)).convert("RGB")

    inputs = processor(images=image, return_tensors="pt").to(DEVICE)

    with torch.no_grad():
        outputs = model(**inputs)
        logits = outputs.logits
        pred_idx = logits.argmax(-1).item()
        label = model.config.id2label[pred_idx]

    return {"predicted_class": label, "class_index": pred_idx}

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

