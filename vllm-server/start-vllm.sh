#!/usr/bin/env bash

vllm serve $HOME/.cache/huggingface/hub/models--unsloth--Qwen3-1.7B-unsloth-bnb-4bit/snapshots/a972f4304f86a0a705d35b378d7a7675d58063a7/ --gpu-memory-utilization 0.6 --max-model-len 12000 --api-server-count 2
