package main

import (
	"flag"
	"fmt"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "config.yaml", "config file location, use absolute path or relative path. make sure your current directory has config.yaml file if use default value.")
	flag.Parse()
}

func main() {
	//get config
	var c config
	conf, err := c.getConfig(configFile)
	if err != nil {
		fmt.Println(err.Error())
		flag.Usage()
		return
	}

	//connect to redis
	connMaster, connSlave, err := getRedisConnMasterSlave(conf.RedisMaster, conf.RedisSlave)
	if err != nil {
		fmt.Println("connect to redis server error:", err)
		return
	}
	defer connMaster.Close()
	if connSlave == nil {
		connSlave = connMaster
	} else {
		defer connMaster.Close()
	}

	var answer string

	//find keys
	keys, err := findKeys(connSlave, conf.Keys, conf.IterNum)
	if err != nil {
		fmt.Println("scan keys error:", err)
		return
	}
	fmt.Println("Show Matched Keys Detail? [y or n]")
	fmt.Scan(&answer)
	if answer == "y" {
		fmt.Println("Matched Keys:", keys)
	}

	//save data
	fmt.Println("Save Data To data.txt Before Delete Keys? [y or n]")
	fmt.Scan(&answer)
	if answer == "y" {
		storeData(connSlave, keys, "data.txt", conf.FetchTypeNum, conf.FetchDataNum)
	}

	//delete keys
	fmt.Println("Delete Match Keys? [y or n]")
	fmt.Scan(&answer)
	if answer == "y" {
		deleteKeys(connMaster, keys, conf.DeleteNum)
	}
	fmt.Println("Script Finish.")
}
