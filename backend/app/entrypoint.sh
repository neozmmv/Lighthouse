#!/bin/sh
HOSTNAME_FILE="/var/lib/tor/hidden_service/hostname"
TIMEOUT=120
elapsed=0

echo "Waiting for Tor hidden service hostname..."
while [ ! -f "$HOSTNAME_FILE" ]; do
    if [ $elapsed -ge $TIMEOUT ]; then
        echo "Timeout waiting for Tor hostname, using fallback: $MINIO_PUBLIC_ENDPOINT"
        break
    fi
    sleep 1
    elapsed=$((elapsed + 1))
done

if [ -f "$HOSTNAME_FILE" ]; then
    ONION=$(cat "$HOSTNAME_FILE" | tr -d '[:space:]')
    export MINIO_PUBLIC_ENDPOINT="http://${ONION}"
    echo "Tor onion address detected: $MINIO_PUBLIC_ENDPOINT"
fi

exec fastapi run main.py --host 0.0.0.0 --port 8000
