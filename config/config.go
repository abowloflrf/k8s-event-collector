package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Log       string
	Receivers struct {
		ElasticSearch *ElasticSearch
		Other         *OtherTarget
	}
}

type ElasticSearch struct {
	Addresses []string
	Index     string
}

type OtherTarget struct {
	Foo string
}

var C *Config

func InitConf(configFile string) {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("/etc/eventsdispathcer")
		viper.AddConfigPath("$HOME/.config/eventsdispathcer")
		viper.AddConfigPath(".")
	}
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatalf("read config file: %v", err)
	}
	if err := viper.Unmarshal(&C); err != nil {
		logrus.Fatalf("unmarshall config to struct: %v", err)
	}
}
