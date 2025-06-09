#!/bin/sh

# Check if yq is installed
if ! command -v yq &> /dev/null; then
    echo "yq is not installed. Please install it first."
    exit 1
fi

CONFIG_FILE_PATH="/app/config.dist.yml"

if [[ -n "$AUTH_KEYS" ]]; then
    AUTH_KEYS_ARRAY=$(echo "$AUTH_KEYS" | tr ',' '\n' | sed -e 's/^ *//; s/ *$//' | sed -e 's/^/"/' -e 's/$/"/' | tr '\n' ',' | sed -e 's/,$//')
    FORMATTED_AUTH_KEYS="[${AUTH_KEYS_ARRAY}]"
    yq e -i ".api_auth_keys = ${FORMATTED_AUTH_KEYS}" $CONFIG_FILE_PATH
fi

if [[ -n "$PORT" ]]; then
    yq e -i ".api.port = ${PORT}" $CONFIG_FILE_PATH
fi

if [[ -n "$HEALTH_PORT" ]]; then
    yq e -i ".api.health.port = ${HEALTH_PORT}" $CONFIG_FILE_PATH
fi

/app/tls-client-api
