package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type ConfigJson struct {
	Content   string
	Output    string
	Templates string
}

func Read(configJsonPath string) (ConfigJson, error) {
	fd, err := os.Open(configJsonPath)
	if err != nil {
		return ConfigJson{}, fmt.Errorf("failed to open: %s", err)
	}
	defer fd.Close()

	configJson := ConfigJson{}
	decoder := json.NewDecoder(fd)
	if err := decoder.Decode(&configJson); err != nil {
		return ConfigJson{}, fmt.Errorf("failed to parse: %s", err)
	}

	return configJson, nil
}
