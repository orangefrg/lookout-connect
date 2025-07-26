#!/bin/sh
set -e

sed "s/{{BIND_ADDRESS}}/0.0.0.0/" /etc/mosquitto/mosquitto.conf.template > /mosquitto/config/mosquitto.conf

if [ ! -f /mosquitto/config/passwd ]; then
  if [ -z "$MQTT_USERNAME" ] || [ -z "$MQTT_PASSWORD" ]; then
    echo "ERROR: First run requires MQTT_USERNAME and MQTT_PASSWORD."
    exit 1
  fi
  mosquitto_passwd -c -b /mosquitto/config/passwd "$MQTT_USERNAME" "$MQTT_PASSWORD"
  chown mosquitto: /mosquitto/config/passwd
  echo "Mosquitto password file created for user: $MQTT_USERNAME"
fi

mosquitto -c /mosquitto/config/mosquitto.conf &

echo "Waiting for Mosquitto port to be available..."
while ! nc -z 127.0.0.1 1883; do
    sleep 0.1
done
echo "Mosquitto is ready."

exec /usr/local/bin/lookout-connect --config /etc/lookout-connect/config.yaml