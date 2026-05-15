package utilities

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func MustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("Error marshaling: %v", err)
	}
	return data
}

func ReadJson(filePath string) ([]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(byteValue) >= 3 && byteValue[0] == 0xEF && byteValue[1] == 0xBB && byteValue[2] == 0xBF {
		byteValue = byteValue[3:]
	}

	var data []interface{}
	if err := json.Unmarshal(byteValue, &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data, nil
}
