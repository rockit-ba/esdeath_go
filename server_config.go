package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var Conf *config

const confFilePath = "serverConf.yaml"

func init() {
	err := configInit()
	if err != nil {
		log.Error(fmt.Sprintf("config init error: %s", err))
		os.Exit(1)
	}
}

func configInit() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get pwd err: %s", err)
	}
	cf, err := os.ReadFile(filepath.Join(cwd, confFilePath))
	if err != nil {
		return fmt.Errorf("read config file error: %s", err)
	}

	Conf = &config{}
	if err := yaml.Unmarshal(cf, &Conf); err != nil {
		return fmt.Errorf("unmarshal config file error: %s", err)
	}
	log.Info("config 初始化完毕", "config", Conf)
	return nil
}

type config struct {
	ServerAddr string `yaml:"server_addr"`
}
