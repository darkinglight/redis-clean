package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
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
var enableSaveKey bool
var enableSaveData bool
var enableDeleteData bool

func init() {
	flag.StringVar(&configFile, "config", "config.yaml", "config file location, use absolute path or relative path. make sure your current directory has config.yaml file if use default value.")
	flag.BoolVar(&enableSaveKey, "enableSaveKey", false, "save matched keys to file.")
	flag.BoolVar(&enableSaveData, "enableSaveData", false, "save data to file.")
	flag.BoolVar(&enableDeleteData, "enableDeleteData", false, "delete data from redis by matched keys.")
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
	if enableSaveKey {
		storeKeys("keys.txt", keys)
	}

	//save data
	if enableSaveData {
		storeData(connSlave, keys, "data.txt", conf.FetchTypeNum, conf.FetchDataNum)
	}

	//delete keys
	if enableDeleteData {
		deleteKeys(connMaster, keys, conf.DeleteNum)
	}
	fmt.Println("Script Finish.")
}

func storeKeys(filePath string, keys []string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	size := len(keys)
	var keyProcess = NewProcess("Store Keys", size)
	for i := 0; i < size; i++ {
		fmt.Fprintln(writer, keys[i])
		keyProcess.Print(i + 1)
	}
	return nil
}
