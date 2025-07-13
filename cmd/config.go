package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type TCPEndpoint struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

type ICMPEndpoint struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

type HTTPEndpoint struct {
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
}

type MonitoringConfig struct {
	NodeName string `yaml:"name"`
	UserName string `yaml:"user"`
	IP       string `yaml:"ip"`
	Port     int    `yaml:"port"`
	IDFile   string `yaml:"id_file"`
}

type ConnectivityConfig struct {
	ICMP []ICMPEndpoint `yaml:"icmp"`
	TCP  []TCPEndpoint  `yaml:"tcp"`
	HTTP []HTTPEndpoint `yaml:"http"`
}

type ScheduleConfig struct {
	IntervalRaw string        `yaml:"interval" default:"4h"`
	SplitterRaw string        `yaml:"splitter" default:"0m"`
	Interval    time.Duration `yaml:"-"`
	Splitter    time.Duration `yaml:"-"`
}

type ExportConfig struct {
	MQTT []MqttConnection `yaml:"mqtt"`
}

type Config struct {
	Nodes        []MonitoringConfig `yaml:"nodes"`
	Connectivity ConnectivityConfig `yaml:"connectivity"`
	Export       ExportConfig       `yaml:"export"`
	Schedule     ScheduleConfig     `yaml:"schedule"`
}

func (m *MonitoringConfig) String() string {
	sb := strings.Builder{}
	sb.WriteString("MonitoringConfig:\n")
	sb.WriteString("NodeName: ")
	sb.WriteString(m.NodeName)
	sb.WriteString(", UserName: ")
	sb.WriteString(m.UserName)
	sb.WriteString(", IP: ")
	sb.WriteString(m.IP)
	sb.WriteString("\n")
	return sb.String()
}

func (e *ExportConfig) String() string {
	sb := strings.Builder{}
	sb.WriteString("Export Config:\n")
	for _, mqtt := range e.MQTT {
		sb.WriteString(mqtt.String())
	}
	return sb.String()
}

func (m *MqttConnection) String() string {
	sb := strings.Builder{}
	sb.WriteString("MQTT:\n")
	sb.WriteString("Name: ")
	sb.WriteString(m.Name)
	sb.WriteString("\n")
	sb.WriteString("Broker: ")
	sb.WriteString(m.Broker)
	sb.WriteString("\n")
	sb.WriteString("Topic: ")
	sb.WriteString(m.Topic)
	sb.WriteString("\n")
	sb.WriteString("ClientID: ")
	sb.WriteString(m.ClientID)
	sb.WriteString("\n")
	sb.WriteString("Qos: ")
	sb.WriteString(strconv.Itoa(m.Qos))
	sb.WriteString("\n")
	sb.WriteString("Retain: ")
	sb.WriteString(strconv.FormatBool(m.Retain))
	sb.WriteString("\n")
	sb.WriteString("Username: ")
	sb.WriteString(m.Username)
	sb.WriteString("\n")
	sb.WriteString("Password: ")
	if len(m.Password) > 4 {
		sb.WriteString(m.Password[:2] + "***" + m.Password[len(m.Password)-2:])
	} else {
		sb.WriteString("***")
	}
	sb.WriteString("\n")
	return sb.String()
}
func (c *ConnectivityConfig) String() string {
	sb := strings.Builder{}
	sb.WriteString("ConnectivityConfig:\n")
	sb.WriteString("ICMP: \n")
	for _, icmp := range c.ICMP {
		sb.WriteString(icmp.Name)
		sb.WriteString(": ")
		sb.WriteString(icmp.Address)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString("TCP:\n")
	for _, tcp := range c.TCP {
		sb.WriteString(tcp.Name)
		sb.WriteString(": ")
		sb.WriteString(tcp.Address)
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(tcp.Port))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString("HTTP:\n")
	for _, http := range c.HTTP {
		sb.WriteString(http.Name)
		sb.WriteString(": ")
		sb.WriteString(http.Address)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (s *ScheduleConfig) String() string {
	sb := strings.Builder{}
	sb.WriteString("ScheduleConfig:\n")
	sb.WriteString("Interval: ")
	sb.WriteString(s.Interval.String())
	sb.WriteString("\n")
	sb.WriteString("Splitter: ")
	sb.WriteString(s.Splitter.String())
	sb.WriteString("\n")
	return sb.String()
}

func (c *Config) String() string {
	sb := strings.Builder{}
	sb.WriteString("Config:\n")
	for _, node := range c.Nodes {
		sb.WriteString(node.String())
	}
	sb.WriteString("\n---\n")
	sb.WriteString(c.Connectivity.String())
	sb.WriteString("\n---\n")
	sb.WriteString(c.Schedule.String())
	sb.WriteString("\n---\n")
	sb.WriteString(c.Export.String())
	sb.WriteString("\n")
	return sb.String()
}

func LoadConfig() (Config, error) {
	configPath := filepath.Join("etc", "lookout-connect", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		return Config{}, fmt.Errorf("config file not found: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %v", err)
	}

	config.Schedule.Interval, err = time.ParseDuration(config.Schedule.IntervalRaw)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse interval: %v", err)
	}

	config.Schedule.Splitter, err = time.ParseDuration(config.Schedule.SplitterRaw)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse splitter: %v", err)
	}

	for _, node := range config.Nodes {
		tcpEndpoint := TCPEndpoint{
			Name:    node.NodeName,
			Address: node.IP,
			Port:    node.Port,
		}
		if slices.Contains(config.Connectivity.TCP, tcpEndpoint) {
			continue
		}
		config.Connectivity.TCP = append(config.Connectivity.TCP, tcpEndpoint)

		icmpEndpoint := ICMPEndpoint{
			Name:    node.NodeName,
			Address: node.IP,
		}
		if slices.Contains(config.Connectivity.ICMP, icmpEndpoint) {
			continue
		}
		config.Connectivity.ICMP = append(config.Connectivity.ICMP, icmpEndpoint)

		httpEndpoint := HTTPEndpoint{
			Name:    node.NodeName,
			Address: fmt.Sprintf("https://%s", node.IP),
		}
		if slices.Contains(config.Connectivity.HTTP, httpEndpoint) {
			continue
		}
		config.Connectivity.HTTP = append(config.Connectivity.HTTP, httpEndpoint)
	}
	for i := range len(config.Export.MQTT) {
		config.Export.MQTT[i].Username = os.Getenv("MQTT_USERNAME")
		config.Export.MQTT[i].Password = os.Getenv("MQTT_PASSWORD")
	}

	log.Printf("Config loaded successfully:\n%v", config.String())
	return config, nil
}
