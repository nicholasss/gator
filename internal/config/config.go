package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	DBURL           string `json:"db_url"`
	CurrentUsername string `json:"current_user_name"`
}

// function to return the path of the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return homeDir + "/.gatorconfig.json", nil
}

// reads config from file and returns
// reads from `~/.gatorconfig.json`
func Read() (Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return Config{}, err
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var loadedConfig Config
	if err = json.Unmarshal(configData, &loadedConfig); err != nil {
		return Config{}, err
	}

	return loadedConfig, nil
}

// writes config to file after setting current user
func (c Config) SetUser(username string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	c.CurrentUsername = username
	configData, err := json.Marshal(c)
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath, configData, 0644)
	if err != nil {
		return err
	}

	return nil
}
