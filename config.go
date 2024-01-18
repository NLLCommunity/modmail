package main

import (
	"errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

var configPath *string = pflag.StringP("config", "C", "", "Path to config file")

func init() {
	pflag.Parse()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")

	viper.SetEnvPrefix("modmail")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var dotConfigPath string
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		dotConfigPath = os.Getenv("XDG_CONFIG_HOME") + "/modmail"
	} else {
		dotConfigPath = os.Getenv("HOME") + "/.config/modmail"
	}
	viper.AddConfigPath(dotConfigPath)

	viper.AddConfigPath("/etc/modmail")
	viper.AddConfigPath(".")

	viper.SetDefault("discord.token", "")
	viper.SetDefault("discord.pub_key", "")
	viper.SetDefault("dev_mode.enabled", false)
	viper.SetDefault("dev_mode.guild", 0)
	viper.SetDefault("http_server.enabled", false)
	viper.SetDefault("http_server.port", 8080)

	viper.AutomaticEnv()

	if *configPath != "" {
		viper.SetConfigFile(*configPath)
	}

	if err := viper.ReadInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			//saveDefaultConfig(*configPath, dotConfigPath)
		} else {
			panic(err)
		}
	}
}

func saveDefaultConfig(configPath, dotPath string) {
	var err error
	if configPath != "" {
		if err = os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			panic(err)
		}
		err = viper.SafeWriteConfigAs(configPath)
	} else {
		if err = os.MkdirAll(dotPath, 0755); err != nil {
			panic(err)
		}
		err = viper.SafeWriteConfig()
	}
	if err != nil {
		panic(err)
	}
}
