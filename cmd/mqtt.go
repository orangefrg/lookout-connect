package main

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttConnection struct {
	Name     string `yaml:"name"`
	Broker   string `yaml:"broker"`
	Topic    string `yaml:"topic"`
	ClientID string `yaml:"client_id"`
	Qos      int    `yaml:"qos"`
	Retain   bool   `yaml:"retain"`
	Username string
	Password string
	Client   mqtt.Client
}

func (m *MqttConnection) Initialize() error {

	opts := mqtt.NewClientOptions()
	opts.AddBroker(m.Broker)
	opts.SetClientID(m.ClientID)
	opts.SetCleanSession(true)
	opts.SetConnectTimeout(30 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	if m.Username != "" {
		opts.SetUsername(m.Username)
		opts.SetPassword(m.Password)
	}

	m.Client = mqtt.NewClient(opts)

	if token := m.Client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker %s: %v", m.Broker, token.Error())
	}

	log.Printf("Successfully connected to MQTT broker: %s", m.Broker)
	return nil
}

func (m *MqttConnection) SendResult(result *MonitoringResult) error {
	log.Printf("[%s] Sending result to MQTT (%s)", result.NodeCfgName, m.Name)
	if m.Client == nil {
		return fmt.Errorf("MQTT client not initialized")
	}
	log.Printf("[%s] Serializing result to JSON", result.NodeCfgName)
	jsonData, err := result.ToJson()
	if err != nil {
		return fmt.Errorf("failed to serialize result to JSON: %v", err)
	}
	log.Printf("[%s] Publishing result to MQTT (%s)", result.NodeCfgName, m.Name)
	token := m.Client.Publish(fmt.Sprintf("%s/%s", m.Topic, result.NodeCfgName), byte(m.Qos), m.Retain, jsonData)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish message to topic %s: %v", m.Topic, token.Error())
	}

	log.Printf("[%s] Successfully published monitoring result to MQTT topic: %s", result.NodeCfgName, m.Topic)
	return nil
}

func (m *MqttConnection) Close() error {
	if m.Client != nil {
		m.Client.Disconnect(250)
	}
	return nil
}

func InitializeMQTTConnections(mqttConnections []MqttConnection) error {
	log.Printf("Initializing %d MQTT connections", len(mqttConnections))
	for i := range len(mqttConnections) {
		if err := mqttConnections[i].Initialize(); err != nil {
			log.Printf("Failed to initialize MQTT connection %s: %v", mqttConnections[i].Name, err)
			continue
		}
		log.Printf("Initialized MQTT connection: %s", mqttConnections[i].Name)
	}
	log.Printf("Initialized %d MQTT connections", len(mqttConnections))
	return nil
}

func CleanupMQTTConnections(mqttConnections []MqttConnection) {
	for _, conn := range mqttConnections {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing MQTT connection %s: %v", conn.Name, err)
		} else {
			log.Printf("Closed MQTT connection: %s", conn.Name)
		}
	}
}
