package main

import (
	"flag"
	"fmt"
)

var configFile string
var redisMasterHost string
var redisMasterPort int
var redisMasterPwd string
var redisSlaveHost string
var redisSlavePort string
var redisSlavePwd string
var keys string
var iterNum int
var fetchTypeNum int
var fetchDataNum int
var deleteNum int
var saveKey bool
var saveData bool
var deleteData bool

func init() {
	flag.StringVar(&configFile, "config", "config.yaml", "config file location, use absolute path or relative path. make sure your current directory has config.yaml file if use default value.")
	flag.BoolVar(&saveKey, "saveKey", false, "if save matched keys to file.")
	flag.BoolVar(&saveData, "saveData", false, "if save data to file.")
	flag.BoolVar(&deleteData, "deleteData", false, "if delete data from redis by matched keys.")
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

	//find keys
	keys, err := findKeys(connSlave, conf.Keys, conf.IterNum)
	if err != nil {
		fmt.Println("scan keys error:", err)
		return
	}
	if saveKey {
		fmt.Println("Matched Keys:", keys)
	}

	//save data
	if saveData {
		storeData(connSlave, keys, "data.txt", conf.FetchTypeNum, conf.FetchDataNum)
	}

	//delete keys
	if deleteData {
		deleteKeys(connMaster, keys, conf.DeleteNum)
	}
	fmt.Println("Script Finish.")
}
