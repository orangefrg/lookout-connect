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
	Address string
	Port    int
}

type MonitoringConfig struct {
	NodeName string `yaml:"name"`
	UserName string `yaml:"user"`
	IP       string `yaml:"ip"`
	Port     int    `yaml:"port"`
	IDFile   string `yaml:"id_file"`
}

type ConnectivityConfig struct {
	ICMP []string      `yaml:"icmp"`
	TCP  []TCPEndpoint `yaml:"tcp"`
}

type ScheduleConfig struct {
	IntervalRaw string        `yaml:"interval" default:"4h"`
	SplitterRaw string        `yaml:"splitter" default:"0m"`
	Interval    time.Duration `yaml:"-"`
	Splitter    time.Duration `yaml:"-"`
}

type Config struct {
	Nodes        []MonitoringConfig `yaml:"nodes"`
	Connectivity ConnectivityConfig `yaml:"connectivity"`
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

func (c *ConnectivityConfig) String() string {
	sb := strings.Builder{}
	sb.WriteString("ConnectivityConfig:\n")
	sb.WriteString("ICMP: \n")
	sb.WriteString(strings.Join(c.ICMP, "\n"))
	sb.WriteString("\n")
	sb.WriteString("TCP:\n")
	for _, tcp := range c.TCP {
		sb.WriteString(tcp.Address)
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(tcp.Port))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
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
	sb.WriteString("\n")
	return sb.String()
}

func LoadConfig() (Config, error) {
	execPath, err := os.Executable()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get executable path: %v", err)
	}

	execDir := filepath.Dir(execPath)
	configPath := filepath.Join(execDir, "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		log.Printf("Config file not found, trying current working directory")
		cwd, err := os.Getwd()
		if err != nil {
			return Config{}, fmt.Errorf("failed to get current working directory: %v", err)
		}
		configPath = filepath.Join(cwd, "config.yaml")
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
		nodeEndpoint := TCPEndpoint{
			Address: node.IP,
			Port:    node.Port,
		}
		if slices.Contains(config.Connectivity.TCP, nodeEndpoint) {
			continue
		}
		config.Connectivity.TCP = append(config.Connectivity.TCP, nodeEndpoint)

		if slices.Contains(config.Connectivity.ICMP, node.IP) {
			continue
		}
		config.Connectivity.ICMP = append(config.Connectivity.ICMP, node.IP)
	}

	log.Printf("Config loaded successfully:\n%v", config.String())
	return config, nil
}
