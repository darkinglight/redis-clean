package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

type config struct {
	RedisMaster redisConfig `yaml:"redisMaster"`
	RedisSlave  redisConfig `yaml:"redisSlave"`
	Keys        string      `yaml:"keys"`
	IterNum     int         `yaml:"iterNum"`
	DeleteNum   int         `yaml:"deleteNum"`
}
type redisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
}

/**
 * 导入配置文件
 */
func (c *config) getConfig(configFile string) (*config, error) {
	//try absolute path
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		//try relative path
		if os.IsNotExist(err) {
			path := filepath.Dir(os.Args[0])
			yamlFile, err = ioutil.ReadFile(path + "/" + configFile)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
