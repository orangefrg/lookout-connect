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
	InitChecks(config)
	timer := time.NewTicker(config.Schedule.Interval)
	defer timer.Stop()
	for range timer.C {
		InitChecks(config)
	}
}

func InitChecks(config Config) []MonitoringResult {
	timer := time.NewTicker(config.Schedule.Splitter)
	defer timer.Stop()
	resultsChan := make(chan MonitoringResult)
	log.Printf("Starting checks")

	wg := sync.WaitGroup{}
	for i, node := range config.Nodes {
		wg.Add(1)
		if i != 0 {
			<-timer.C
			log.Println("Waiting for next node check")
		}
		go func(node MonitoringConfig) {
			defer wg.Done()
			currentResult := node.PerformChecks(config.Connectivity)
			resultsChan <- currentResult
		}(node)
	}

	log.Printf("Waiting for results")
	results := make([]MonitoringResult, 0)
	go func() {
		for i := 0; i < len(config.Nodes); i++ {
			currentResult := <-resultsChan
			log.Println("Received result")
			results = append(results, currentResult)
			log.Println(currentResult.String())
		}
		close(resultsChan)
	}()
	wg.Wait()
	log.Println("Results ready!")
	return results
}
