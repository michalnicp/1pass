package op

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
)

const (
	opConfigPath = ".op"
	opConfigFile = "config"
)

type Config struct {
	LatestSignin string    `json:"latest_signin"`
	Accounts     []Account `json:"accounts"`
}

type Account struct {
	Shorthand  string `json:"shorthand"`
	URL        string `json:"url"`
	Email      string `json:"email"`
	AccountKey string `json:"accountKey"`
	UserUUID   string `json:"userUUID"`
}

func ReadConfig() (*Config, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(user.HomeDir, opConfigPath, opConfigFile)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config Config
	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
