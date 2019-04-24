package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	*viper.Viper
}

func (c Config) MustGetString(key string) string {
	v := c.GetString(key)
	if v == "" {
		log.Fatalf("%s is empty", key)
	}
	return v
}

func New() *Config {
	c := Config{Viper: viper.New()}
	c.BindEnv("CONF_FILE")
	c.SetConfigFile(c.GetString("CONF_FILE"))

	if err := c.ReadInConfig(); err != nil {
		log.Fatalf("Failed to load conf file: %s", err)
	}
	return &c
}
