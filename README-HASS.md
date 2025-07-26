# Configuration Generator

This directory contains scripts to automatically generate `cards.yaml` and `hass.yaml` files from your `config.yaml` configuration.

## Files

- `generate_configs.py` - Python script that reads `config.yaml` and generates the configuration files
- `generate_configs.sh` - Shell script wrapper for easier usage
- `README_generator.md` - This documentation file

## Usage

### Quick Start

1. Make sure you have a `config.yaml` file in the current directory
2. Run the generator:

```bash
./generate_configs.sh
```

Or directly with Python:

```bash
python3 generate_configs.py
```

### Requirements

- Python 3 (with `yaml` module - usually included by default)
- A valid `config.yaml` file

### What it generates

The script reads your `config.yaml` file and generates:

1. **cards.yaml** - Home Assistant dashboard cards configuration
   - Creates a grid layout for each node in your config
   - Includes entity cards for disk usage, check duration, and hostname
   - Adds interconnectivity sensors between all nodes
   - Includes markdown cards for connectivity and login records

2. **hass.yaml** - Home Assistant MQTT sensor configuration
   - Creates MQTT sensors for each node's basic information
   - Adds interconnectivity sensors for TCP and ICMP tests
   - Includes proper templates for data processing

### Configuration Structure

The script expects your `config.yaml` to have a `nodes` section with entries like:

```yaml
nodes:
  - name: "node-name"
    ip: "192.168.1.100"
    port: 22
    user: "username"
    id_file: "/path/to/ssh/key"
```

### Customization

If you need to modify the generation logic:

1. Edit `generate_configs.py`
2. The main functions are:
   - `write_cards_yaml()` - Generates the dashboard cards
   - `write_hass_yaml()` - Generates the MQTT sensors

### Troubleshooting

- **"config.yaml not found"**: Make sure you're in the directory containing your config file
- **"Python 3 is required"**: Install Python 3 on your system
- **YAML parsing errors**: Check that your `config.yaml` is valid YAML syntax

## Integration

After generating the files:

1. Copy `hass.yaml` content to your Home Assistant `configuration.yaml` or include it as a separate file
2. Copy `cards.yaml` content to your Home Assistant dashboard configuration
3. Restart Home Assistant to load the new configuration

The generated configuration will automatically adapt to any changes you make to your `config.yaml` file - just run the generator again to update the files. 