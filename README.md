## Simple monitoring over SSH

#### Capabilities

- Connect to a host via SSH (using a key)
- Check disk usage
- Check last logins
- Check connectivity
  - ICMP Ping
  - Raw TCP
  - Curl
- Scheduling checks (simple intervals)
- Offsetting checks of individual nodes (to reduce load on networks)
- Cross-checking nodes (connectivity between them)

#### Building

`docker build --no-cache -t lookout-mosquitto:latest .`

#### Running

Ensure you have those files:
- `.env` — you can make one from template
- `config.yaml` — you can make one from template, it's pretty self-explanatory
- `docker-compose.yml` — pay attention to logging settings
- `entrypoint.sh`
- `lookout-mosquitto.tar` — in case you haven't loaded it yet

Then generate an ssh key:
`ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa_lookout -C "lookout ssh keys"`

Make sure your public key part is added to every host you want to check (`nodes` in config)

Make deploy script runnable:
`chmod +x deploy.sh`

Launch deploy file
`./deploy.sh ./id_rsa_lookout`

You can check everything in logs:
`docker compose logs -f`
Or check data in MQTT:
`export $(grep -v '^#' .env | xargs)`
`mosquitto_sub -h 127.0.0.1 -p 1883 -u "$MQTT_USERNAME" -P "$MQTT_PASSWORD" -t "#" -v`

#### Reading

Connect with any MQTT client to a broker inside container (port is specified in the compose file, credentials are in `.env`).
The messages are retained by default and have QoS 0. You can change it in config.