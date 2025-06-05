#!/bin/bash

# NOTE: You can run `./cleanup.sh` before this script.

# Create a custom network.
docker network create trpc-network

# Run the Prometheus container using the official image.
docker run -d --name prometheus --network trpc-network -p 9090:9090 \
  --cpuset-cpus="0" \
  -v $(pwd)/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus

# Run the Grafana container using the official image with provisioning.
docker run -d --name grafana --network trpc-network -p 3000:3000 \
  --cpuset-cpus="1" \
  -v $(pwd)/grafana/provisioning:/etc/grafana/provisioning \
  grafana/grafana

server_tag=trpc-robust-server:latest
client_tag=trpc-robust-client:latest

# Build the server and client
cd server && go build -o server . && docker build -t $server_tag . && cd -
cd client && go build -o client . && docker build -t $client_tag . && cd -

# Run the server on the custom network.
docker run -itd --name robust-server --network trpc-network \
  -p 8000:8000 -p 9028:9028 -p 8090:8090 \
  --cpuset-cpus="2,3" \
  -v $(pwd)/server:/app \
  $server_tag

# Run the client on the custom network.
docker run -itd --name robust-client --network trpc-network \
  -p 9029:9029 -p 8092:8092 \
  --cpuset-cpus="4-7" \
  -v $(pwd)/client:/app \
  $client_tag

# Print the status of the containers
docker ps
