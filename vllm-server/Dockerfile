FROM vllm/vllm-openai:v0.11.0

COPY huggingface-cache /root/.cache/huggingface

CMD [ \
    "--gpu-memory-utilization", "0.7", \
    "--model", "unsloth/Qwen3-1.7B-unsloth-bnb-4bit", \
    "--max-model-len", "4096" \
]
