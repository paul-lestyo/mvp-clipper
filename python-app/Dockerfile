FROM python:3.10-slim

WORKDIR /app

# Install system dependencies required for OpenCV and MediaPipe
RUN apt-get update && apt-get install -y --no-install-recommends \
    libglib2.0-0 \
    libsm6 \
    libxext6 \
    libxrender-dev \
    libgl1 \
    curl \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Unbuffered output for better logging
ENV PYTHONUNBUFFERED=1

CMD ["python", "main.py"]
