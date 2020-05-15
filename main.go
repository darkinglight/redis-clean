package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
    "time"
    "sync"
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

	var searchProcess = NewProcess(fmt.Sprintf("Search %s", conf.Keys))

    var wg sync.WaitGroup
    wg.Add(2)

	//find keys
    keysChannel := make(chan string, 100)
	go findKeys(connSlave, conf.Keys, conf.IterNum, keysChannel, searchProcess, &wg)

    saveKeyChannel := make(chan string)
    saveDataChannel := make(chan string)
    deleteChannel := make(chan string)
    go func() {
        defer wg.Done()
        for key := range keysChannel {
	        if enableSaveKey {
                saveKeyChannel <- key
	        }
	        if enableSaveData {
                saveDataChannel <- key
	        }
	        if enableDeleteData {
                deleteChannel <- key
	        }
        }
        close(saveKeyChannel)
        close(saveDataChannel)
        close(deleteChannel)
    }()

    //save keys
	if enableSaveKey {
	    go storeKeys("keys.txt", saveKeyChannel)
	}

	//save data
	if enableSaveData {
	    go storeData(connSlave, saveDataChannel, "data.txt", conf.FetchTypeNum, conf.FetchDataNum)
	}

	//delete keys
	if enableDeleteData {
	    go deleteKeys(connMaster, deleteChannel, conf.DeleteNum, searchProcess, &wg)
	}

    //show process
    done := make(chan bool)
    ticker := time.NewTicker(1000 * time.Millisecond)
    go func() {
        for {
            select {
            case <-done:
                return
            case <-ticker.C:
                searchProcess.Print()
            }
        }
    }()

    wg.Wait()
    done <- true
	fmt.Println("Script Finish.")
}

func storeKeys(filePath string, keys <-chan string) {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
        fmt.Println(err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

    for key := range keys {
		fmt.Fprintln(writer, key)
	}
}
