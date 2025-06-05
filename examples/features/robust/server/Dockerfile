FROM centos:latest

WORKDIR /app

# Copy the pre-built server binary.
COPY server /usr/local/bin/server
# Copy the framework configuration.
COPY trpc_go.yaml /etc/trpc/

# Create an entrypoint script using a here-document within a RUN command.
COPY <<EOF /usr/local/bin/entrypoint.sh
#!/bin/bash
# Prepend the default arguments to the provided ones.
set -- -conf /etc/trpc/trpc_go.yaml "\$@"
# Execute the server binary with all arguments.
exec /usr/local/bin/server "\$@"
EOF

RUN chmod +x /usr/local/bin/entrypoint.sh

# Expose service port.
EXPOSE 8000
# Expose admin port.
EXPOSE 9028

# Set the entrypoint to the entrypoint script.
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD []
