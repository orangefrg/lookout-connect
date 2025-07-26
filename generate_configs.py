#!/usr/bin/env python3
"""
Script to generate cards.yaml and hass.yaml files based on config.yaml
"""

import yaml
import sys
from pathlib import Path

def load_config(config_file):
    """Load the config.yaml file"""
    with open(config_file, 'r') as f:
        return yaml.safe_load(f)

def write_cards_yaml(config, output_file):
    """Write cards.yaml content directly"""
    nodes = config.get('nodes', [])
    
    with open(output_file, 'w') as f:
        f.write("""  - type: sections
    max_columns: 4
    icon: mdi:lan-pending
    path: vps
    title: VPS
    sections:
""")
        
        for node in nodes:
            node_name = node['name']
            display_name = node_name.replace('-', ' ').title()
            
            f.write(f"""      - type: grid
        cards:
          - type: heading
            heading: {display_name}
            heading_style: title
          - type: entities
            entities:
              - entity: sensor.lookout_{node_name}_disk_usage
                name: Disk usage
                secondary_info: last-changed
              - entity: sensor.lookout_{node_name}_last_check_duration
                secondary_info: last-updated
                icon: mdi:progress-clock
                name: Last Check
              - entity: sensor.lookout_{node_name}_hostname
                icon: mdi:badge-account-horizontal-outline
                name: Hostname
""")
            
            # Add interconnect entities
            interconnect_entities = []
            for other_node in nodes:
                if other_node['name'] != node_name:
                    other_display_name = other_node['name'].replace('-', ' ').title()
                    interconnect_entities.extend([
                        f"""              - entity: sensor.interconnect_lookout_{node_name}_to_{other_node["name"]}_icmp
                name: {other_display_name} (ICMP)
                secondary_info: last-changed""",
                        f"""              - entity: sensor.interconnect_lookout_{node_name}_to_{other_node["name"]}_tcp
                name: {other_display_name} (TCP)
                secondary_info: last-changed"""
                    ])
            
            if interconnect_entities:
                f.write("          - type: entities\n            entities:\n")
                for entity in interconnect_entities:
                    f.write(f"{entity}\n")
            
            # Add connectivity markdown card
            f.write(f"""          - type: markdown
            title: Connectivity ({display_name})
            content: >-
              {{% set conn = state_attr('sensor.lookout_{node_name}_login_records', 'connectivity') or %{{}} %}}
              {{% if conn %}}
              {{% for host, checks in conn.items() %}}
              **{{{{ host | trim }}}}**
                {{%- if 'http' in checks %}}
              
                  - HTTP:
                    {{%- for test in checks.http %}}
                      {{{{ '✅ ' ~ test.code if test.status and test.code < 400 else '❌ DOWN' }}}}{{{{ ',' if not loop.last else '' }}}}
                    {{%- endfor %}}
                {{%- endif %}}
                {{%- if 'icmp' in checks %}}
              
                  - ICMP:
                    {{%- for test in checks.icmp %}}
                      {{{{ '✅ OK' if test.status else '❌ FAIL' }}}}{{{{ ',' if not loop.last else '' }}}}
                    {{%- endfor %}}
                {{%- endif %}}
                {{%- if 'tcp' in checks %}}
              
                  - TCP:
                    {{%- for test in checks.tcp %}}
                      {{{{ test.port }}}} {{{{ '✅' if test.status else '❌' }}}}{{{{ ',' if not loop.last else '' }}}}
                    {{%- endfor %}}
                {{%- endif %}}
              {{% endfor %}}
              {{% else %}}
              No connectivity data available.
              {{% endif %}}
          - type: markdown
            title: Login Records ({display_name})
            content: >-
              {{% set logins = state_attr('sensor.lookout_{node_name}_login_records', 'login_records') or [] %}}
              {{% if logins %}}
              **Recent logins (up to 5):**
              {{% set count = 0 %}}
              {{% for l in logins | sort(attribute='login_time', reverse=True) %}}
                {{% if count < 5 %}}
              - {{{{ l.username }}}} ({{{{ l.ip or 'local' }}}}) — {{{{ l.login_time }}}} → {{{{ l.logout_time }}}}
                  {{% set count = count + 1 %}}
                {{% endif %}}
              {{% endfor %}}
              {{% else %}}
              No login data available.
              {{% endif %}}
""")

def write_hass_yaml(config, output_file):
    """Write hass.yaml content directly"""
    nodes = config.get('nodes', [])
    
    with open(output_file, 'w') as f:
        f.write("mqtt:\n  sensor:\n\n")
        
        for node in nodes:
            node_name = node['name']
            
            # Basic sensors for each node
            f.write(f"""    # {node_name}
    - name: "Lookout: {node_name} Last Check Time"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ value_json.last_check_time }}}}"
      icon: mdi:clock-outline

    - name: "Lookout: {node_name} Hostname"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ value_json.hostname }}}}"

    - name: "Lookout: {node_name} User"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ value_json.user }}}}"
    
    - name: "Lookout: {node_name} Disk Usage"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ value_json.disk_usage }}}}"
      unit_of_measurement: "%"
      icon: mdi:harddisk

    - name: "Lookout: {node_name} Free Space"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ (value_json.free_space | float / 1024 / 1024 / 1024) | round(1) }}}}"
      unit_of_measurement: "GB"

    - name: "Lookout: {node_name} Total Space"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ (value_json.total_space | float / 1024 / 1024 / 1024) | round(1) }}}}"
      unit_of_measurement: "GB"

    - name: "Lookout: {node_name} Last Check Duration"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ value_json.check_duration | round(1) }}}}"
      unit_of_measurement: "s"

    - name: "Lookout: {node_name} Login Records"
      state_topic: "vps-monitoring/{node_name}"
      value_template: >
        {{% set recs = value_json.login_records | default([]) %}}
        {{% if recs | length > 0 %}}
          {{% set sorted = recs | sort(attribute='login_time', reverse=True) %}}
          {{{{ sorted[0].login_time }}}}
        {{% else %}}
          unknown
        {{% endif %}}
      json_attributes_topic: "vps-monitoring/{node_name}"
      json_attributes_template: >
        {{% set recs = value_json.login_records | default([]) %}}
        {{% set sorted = recs | sort(attribute='login_time', reverse=True) %}}
        {{% set ns = namespace(items=[]) %}}
        {{% for r in sorted %}}
          {{% if loop.index <= 10 %}}
            {{% set ns.items = ns.items + [r] %}}
          {{% endif %}}
        {{% endfor %}}

        {{% set conn = value_json.connectivity | default({{}}) %}}
        {{% if conn is not mapping %}}
          {{% set conn = {{}} %}}
        {{% endif %}}

        {{{{ {{"login_records": ns.items, "connectivity": conn}} | tojson }}}}

    - name: "Lookout: {node_name} Connectivity"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ value_json.connectivity }}}}"

""")
            
            # Add interconnect sensors for each other node
            for other_node in nodes:
                if other_node['name'] != node_name:
                    other_name = other_node['name']
                    f.write(f"""    - name: "Interconnect Lookout: {node_name} to {other_name} (TCP)"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ value_json.connectivity.{other_name}.tcp[0].status }}}}"
      icon: mdi:lan-connect
      
    - name: "Interconnect Lookout: {node_name} to {other_name} (ICMP)"
      state_topic: "vps-monitoring/{node_name}"
      value_template: "{{{{ value_json.connectivity.{other_name}.icmp[0].status }}}}"
      icon: mdi:lan-connect

""")

def main():
    """Main function"""
    config_file = 'config.yaml'
    
    if not Path(config_file).exists():
        print(f"Error: {config_file} not found")
        sys.exit(1)
    
    try:
        config = load_config(config_file)
    except Exception as e:
        print(f"Error loading config: {e}")
        sys.exit(1)
    
    # Generate cards.yaml
    try:
        write_cards_yaml(config, 'cards.yaml')
        print("Generated cards.yaml successfully")
    except Exception as e:
        print(f"Error generating cards.yaml: {e}")
        sys.exit(1)
    
    # Generate hass.yaml
    try:
        write_hass_yaml(config, 'hass.yaml')
        print("Generated hass.yaml successfully")
    except Exception as e:
        print(f"Error generating hass.yaml: {e}")
        sys.exit(1)
    
    print("All files generated successfully!")

if __name__ == '__main__':
    main() 