FROM golang:1.22

WORKDIR /app

# Install ffmpeg, yt-dlp, dan fonts
RUN apt-get update && apt-get install -y \
    ffmpeg \
    python3 \
    python3-pip \
    fonts-liberation \
    fonts-dejavu-core \
    fontconfig \
    && pip3 install --break-system-packages yt-dlp \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && fc-cache -f -v

COPY . .

RUN go mod download
RUN go build -o server ./cmd/server

CMD ["./server"]
