package main

import (
	"encoding/json"
)

func (r *MonitoringResult) ToJson() (string, error) {
	jsonData, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
