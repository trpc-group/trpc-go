FROM centos:latest

WORKDIR /app

# Copy the pre-built client binary.
COPY client /usr/local/bin/client
# Copy the framework configuration.
COPY trpc_go.yaml /etc/trpc/

# Create an entrypoint script.
COPY <<EOF /usr/local/bin/entrypoint.sh
#!/bin/bash
# Prepend the default arguments to the provided ones.
set -- -conf /etc/trpc/trpc_go.yaml "$@"

# Execute the client binary with all arguments.
exec /usr/local/bin/client "$@"
EOF

# Make the entrypoint script executable.
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set the entrypoint to the entrypoint script.
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD []
