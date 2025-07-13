package main

import (
	"log"
	"sync"
	"time"
)

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Config loaded successfully")

	RunSchedule(config)
}

func RunSchedule(config Config) {
	log.Println("Running schedule!")
	err := InitializeMQTTConnections(config.Export.MQTT)
	if err != nil {
		log.Printf("Warning: Failed to initialize MQTT connections: %v", err)
		return
	}
	InitChecks(config)
	log.Println("Checks finished! Cleaning up...")
	CleanupMQTTConnections(config.Export.MQTT)

	timer := time.NewTicker(config.Schedule.Interval)
	defer timer.Stop()
	mtx := sync.Mutex{}
	for range timer.C {
		if mtx.TryLock() {
			err := InitializeMQTTConnections(config.Export.MQTT)
			if err != nil {
				log.Printf("Warning: Failed to initialize MQTT connections: %v", err)
				mtx.Unlock()
				continue
			}
			InitChecks(config)
			log.Println("Checks finished! Cleaning up...")
			CleanupMQTTConnections(config.Export.MQTT)
			mtx.Unlock()
		} else {
			log.Println("Skipping schedule: last one haven't finished yet")
			continue
		}
	}
}

func InitChecks(config Config) {
	timer := time.NewTicker(config.Schedule.Splitter)
	defer timer.Stop()
	resultsChan := make(chan MonitoringResult)
	log.Printf("Starting checks")

	wgChecks := sync.WaitGroup{}
	for i, node := range config.Nodes {
		wgChecks.Add(1)
		if i != 0 {
			<-timer.C
			log.Println("Waiting for next node check")
		}
		go func(node MonitoringConfig) {
			defer wgChecks.Done()
			currentResult := node.PerformChecks(config.Connectivity)
			resultsChan <- currentResult
		}(node)
	}

	log.Printf("Waiting for results")
	for i := 0; i < len(config.Nodes); i++ {
		currentResult := <-resultsChan
		log.Println("Received result")
		for _, mqtt := range config.Export.MQTT {
			err := mqtt.SendResult(&currentResult)
			if err != nil {
				log.Printf("Warning: Failed to send result to MQTT %s: %v", mqtt.Name, err)
			}
		}
	}
	close(resultsChan)
	wgChecks.Wait()
	log.Println("Checks finished!")
}
