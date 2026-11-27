FROM mcr.microsoft.com/devcontainers/base:bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
      build-essential ca-certificates curl dnsutils git jq netcat-openbsd \
      nmap python3 rsync unzip vim whois zip \
    && rm -rf /var/lib/apt/lists/*
