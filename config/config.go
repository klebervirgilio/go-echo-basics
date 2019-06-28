package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	*viper.Viper
}

func (c Config) MustGetString(key string) string {
	v := c.GetString(key)
	if v == "" {
		panic(fmt.Sprintf("%s is empty", key))
	}
	return v
}

func New() *Config {
	c := &Config{Viper: viper.New()}
	c.BindEnv("CONF_FILE")
	c.BindEnv("mailChecker.accessKey", "MAIL_CHECKER_ACCESS_KEY")
	c.SetConfigFile(c.MustGetString("CONF_FILE"))

	if err := c.ReadInConfig(); err != nil {
		log.Fatalf("Failed to load conf file: %s", err)
	}
	return c
}
