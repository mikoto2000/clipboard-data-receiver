FROM mcr.microsoft.com/devcontainers/go:1-1.22-bookworm
LABEL maintainer "mikoto2000 <mikoto2000@gmail.com>"

RUN apt update \
    && apt install -y libx11-dev \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

