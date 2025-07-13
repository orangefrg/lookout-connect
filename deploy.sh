#!/usr/bin/env bash
set -euo pipefail

# Usage help
if [ "$#" -ne 1 ]; then
  echo "Usage: $0 path/to/id_rsa"
  exit 1
fi

ID_RSA_PATH="$1"
IMAGE_TAR="lookout-mosquitto.tar"
IMAGE_NAME="lookout-mosquitto:latest"
SSH_DIR="/opt/ssh_keys"
MOSQUITTO_DATA="/opt/mosquitto_data"
MOSQUITTO_CONFIG="/opt/mosquitto_config"
CONFIG_YAML="config.yaml"


# Check for yq, install if missing
if ! command -v yq >/dev/null 2>&1; then
  echo "yq not found. Installing yq..."
  YQ_URL="https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64"
  sudo wget -q -O /usr/local/bin/yq "$YQ_URL"
  sudo chmod +x /usr/local/bin/yq
  if ! command -v yq >/dev/null 2>&1; then
    echo "ERROR: yq installation failed."
    exit 1
  fi
  echo "yq installed successfully."
else
  echo "yq is already installed."
fi

# Check required files
for f in .env docker-compose.yml "$CONFIG_YAML"; do
  if [ ! -f "$f" ]; then
    echo "ERROR: Required file '$f' is missing."
    exit 1
  fi
done

# Check Docker image
if ! docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
  echo "Docker image '$IMAGE_NAME' not found. Loading from tar..."
  docker load -i "$IMAGE_TAR"
else
  echo "Docker image '$IMAGE_NAME' already loaded. Skipping load."
fi

# Create directories
echo "Creating directories..."
mkdir -p "$SSH_DIR" "$MOSQUITTO_DATA" "$MOSQUITTO_CONFIG"
chmod 700 "$SSH_DIR"
chmod 755 "$MOSQUITTO_DATA" "$MOSQUITTO_CONFIG"

# Prepare known_hosts
KNOWN_HOSTS_TMP="./known_hosts.tmp"
: > "$KNOWN_HOSTS_TMP"

echo "Fetching SSH host keys from nodes in $CONFIG_YAML..."

NODES_COUNT=$(yq e '.nodes | length' "$CONFIG_YAML")

for i in $(seq 0 $((NODES_COUNT - 1))); do
  NAME=$(yq e ".nodes[$i].name" "$CONFIG_YAML")
  IP=$(yq e ".nodes[$i].ip" "$CONFIG_YAML")
  PORT=$(yq e ".nodes[$i].port" "$CONFIG_YAML")

  echo " - $NAME ($IP:$PORT)..."
  ssh-keyscan -p "$PORT" -H "$IP" >> "$KNOWN_HOSTS_TMP" 2>/dev/null
done

# Copy known_hosts
echo "Deploying known_hosts..."
cp "$KNOWN_HOSTS_TMP" "$SSH_DIR/known_hosts"
chmod 644 "$SSH_DIR/known_hosts"
rm -f "$KNOWN_HOSTS_TMP"

# Backup old id_rsa
if [ -f "$SSH_DIR/id_rsa_lookout-connect" ]; then
  echo "Backing up existing SSH key..."
  cp "$SSH_DIR/id_rsa_lookout-connect" "$SSH_DIR/id_rsa_lookout-connect.bak.$(date +%s)"
fi

# Copy id_rsa
echo "Copying id_rsa..."
cp "$ID_RSA_PATH" "$SSH_DIR/id_rsa_lookout-connect"
chmod 600 "$SSH_DIR/id_rsa_lookout-connect"

# Stop old containers
docker compose down || true

# Start containers
echo "Starting deployment..."
docker compose up -d

echo "Deployment complete!"
