#!/bin/bash

# File path
FILE="trpc_go.yaml"

# Check if the current scope is local.
if grep -q "^  scope: \"local\"" "$FILE"; then
    # Current scope is local, change to remote.
    before="local"
    after="remote"
    sed -i 's/^  scope: "local"/  # scope: "local"/' "$FILE"
    sed -i 's/^  # scope: "remote"/  scope: "remote"/' "$FILE"
else
    # Current scope is remote, change to local.
    before="remote"
    after="local"
    sed -i 's/^  scope: "remote"/  # scope: "remote"/' "$FILE"
    sed -i 's/^  # scope: "local"/  scope: "local"/' "$FILE"
fi

echo "YAML configuration toggled in $FILE from '$before' to '$after'"
