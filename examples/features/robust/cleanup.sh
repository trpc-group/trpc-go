#!/bin/bash

# Stop and remove the Prometheus container if it exists.
docker stop prometheus 2>/dev/null
docker rm -f prometheus 2>/dev/null

# Stop and remove the Grafana container if it exists.
docker stop grafana 2>/dev/null
docker rm -f grafana 2>/dev/null

# Stop and remove the server container if it exists.
docker stop robust-server 2>/dev/null
docker rm -f robust-server 2>/dev/null

# Stop and remove the client container if it exists.
docker stop robust-client 2>/dev/null
docker rm -f robust-client 2>/dev/null

# Remove the custom network if it exists.
docker network rm trpc-network 2>/dev/null

# Optionally, remove the images if you want a clean state.
# Uncomment the following lines if you want to remove images as well.
# docker rmi trpc-robust-server:latest 2>/dev/null
# docker rmi trpc-robust-client:latest 2>/dev/null
# docker rmi prom/prometheus 2>/dev/null
# docker rmi grafana/grafana 2>/dev/null

echo "Cleanup complete."
