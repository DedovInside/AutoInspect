FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        libgl1 \
        libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY app ./app
COPY scripts ./scripts
COPY proto ./proto
COPY .env.example ./

RUN python -m grpc_tools.protoc \
    -I ./proto \
    --python_out=./app/generated \
    ./proto/analysis/v1/request.proto \
    ./proto/analysis/v1/result.proto

EXPOSE 8081

CMD ["python", "-m", "app.kafka.worker"]
