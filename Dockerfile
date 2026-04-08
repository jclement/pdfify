# Multi-stage Dockerfile for building pdfify from source.
# The final image includes both the Go binary and all conversion tools.
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /pdfify ./cmd/pdfify

FROM node:20-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update -qq && \
    apt-get install -y --no-install-recommends \
        pandoc \
        texlive-latex-recommended \
        texlive-latex-extra \
        texlive-fonts-recommended \
        texlive-fonts-extra \
        texlive-xetex \
        texlive-plain-generic \
        lmodern \
        librsvg2-bin \
        chromium \
        ca-certificates \
        fonts-liberation \
        fonts-roboto \
        fonts-roboto-unhinted \
        fonts-noto-color-emoji \
        wget \
        fontconfig \
    && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /usr/share/fonts/truetype/roboto-mono && \
    for style in Regular Bold Italic BoldItalic Medium MediumItalic Light LightItalic; do \
        wget -q --tries=3 "https://github.com/googlefonts/RobotoMono/raw/v3.001/fonts/ttf/RobotoMono-${style}.ttf" \
             -O "/usr/share/fonts/truetype/roboto-mono/RobotoMono-${style}.ttf"; \
    done && \
    fc-cache -f
RUN npm install -g @mermaid-js/mermaid-cli
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true
ENV PUPPETEER_EXECUTABLE_PATH=/usr/bin/chromium
ENV CHROME_PATH=/usr/bin/chromium
COPY --from=build /pdfify /usr/local/bin/pdfify
WORKDIR /workspace
ENTRYPOINT ["pdfify"]
