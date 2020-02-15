package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"os"
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
	fmt.Println("Save Data Before Delete Keys? [y or n]")
	fmt.Scan(&answer)
	if answer == "y" {
		storeData(connSlave, keys, "data.txt")
	}

	//delete keys
	fmt.Println("Delete Match Keys? [y or n]")
	fmt.Scan(&answer)
	if answer == "y" {
		deleteKeys(connMaster, keys, conf.DeleteNum)
	}
	fmt.Println("Script Finish.")
}

/**
 * 连接redis主从库
 */
func getRedisConnMasterSlave(confMaster redisConfig, confSlave redisConfig) (redis.Conn, redis.Conn, error) {
	connMaster, err := getRedisConn(confMaster)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("connect to redis master suucess.")

	var connSlave redis.Conn
	if confSlave.Host == "" {
		connSlave = nil
	} else {
		connSlave, err = getRedisConn(confSlave)
		if err != nil {
			return nil, nil, err
		}
		fmt.Println("connect to redis slave suucess.")
	}

	return connMaster, connSlave, nil
}

/**
 * 连接redis
 */
func getRedisConn(conf redisConfig) (redis.Conn, error) {
	redisAddress := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	conn, err := redis.Dial("tcp", redisAddress)
	if err != nil {
		return nil, err
	}
	if conf.Password != "" {
		if _, err := conn.Do("AUTH", conf.Password); err != nil {
			return nil, err
		}
	}
	return conn, nil
}

/**
 * 进度
 */
type process struct {
	Name  string
	Value int
}

func (p *process) Print(currentProcess int) {
	if currentProcess > 100 {
		currentProcess = 100
	}
	if currentProcess > p.Value {
		p.Value = currentProcess
		fmt.Printf("%s Process: %d%%\r", p.Name, p.Value)
		if currentProcess == 100 {
			fmt.Println()
		}
	}
}

/**
 * 根据正则表达式查询redis key
 */
func findKeys(conn redis.Conn, pattern string, iterNum int) ([]string, error) {
	//get total key count
	totalCount, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		return nil, err
	}

	var searchProcess = process{"Search Key", 0}
	fmt.Printf("Search Key By Pattern %s Start:\n", pattern)

	iter := 0
	var keysMatch, keys []string
	for i := 1; ; i++ {
		if arr, err := redis.MultiBulk(conn.Do("SCAN", iter, "MATCH", pattern, "COUNT", iterNum)); err != nil {
			return nil, err
		} else {
			iter, _ = redis.Int(arr[0], nil)
			keys, _ = redis.Strings(arr[1], nil)
		}
		if len(keys) > 0 {
			keysMatch = append(keysMatch, keys...)
		}
		searchProcess.Print(i * iterNum * 100 / totalCount)
		if iter == 0 {
			break
		}
	}
	fmt.Printf("Search Key Finish. Total Search Key Number: %d, Match Key Number: %d\n", totalCount, len(keysMatch))

	return keysMatch, nil
}

/**
 * 删除redis key
 */
func deleteKeys(conn redis.Conn, keys []string, nums int) error {
	var deleteProcess = process{"Delete Keys", 0}
	fmt.Println("Delete Keys Start:")

	size := len(keys)
	var totalDeleteNum int
	var part []string
	for i := 0; i*nums < size; i++ {
		if (i+1)*nums > size {
			part = keys[i*nums:]
		} else {
			part = keys[i*nums : (i+1)*nums]
		}
		deleteNum, err := redis.Int(conn.Do("DEL", redis.Args{}.AddFlat(part)...))
		if err != nil {
			return err
		}
		totalDeleteNum += deleteNum
		deleteProcess.Print(totalDeleteNum * 100 / size)
	}
	fmt.Printf("Delete Keys Finish. Match Size:%d; Delete Size:%d\n", size, totalDeleteNum)
	return nil
}

/**
 * 保存redis数据
 */
func storeData(conn redis.Conn, keys []string, filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	var storeProcess = process{"Store Data", 0}
	fmt.Println("Store Data Start:")

	size := len(keys)
	var keyType string
	var data string
	var dataSlice []string
	var dataMap map[string]string
	for i := 0; i < size; i++ {
		storeProcess.Print((i + 1) * 100 / size)
		if keyType, err = redis.String(conn.Do("TYPE", keys[i])); err != nil {
			return err
		}
		switch keyType {
		case "string":
			if data, err = redis.String(conn.Do("GET", keys[i])); err != nil {
				return err
			}
			fmt.Fprintln(writer, keys[i], data)
		case "list":
			if dataSlice, err = redis.Strings(conn.Do("LRANGE", keys[i], 0, -1)); err != nil {
				return err
			}
			fmt.Fprintln(writer, keys[i], dataSlice)
		case "zset":
			if dataMap, err = redis.StringMap(conn.Do("ZRANGE", keys[i], 0, -1, "WITHSCORES")); err != nil {
				return err
			}
			fmt.Fprintln(writer, keys[i], dataMap)
		case "hash":
			if dataMap, err = redis.StringMap(conn.Do("HGETALL", keys[i])); err != nil {
				return err
			}
			fmt.Fprintln(writer, keys[i], dataMap)
		case "set":
			if dataSlice, err = redis.Strings(conn.Do("SMEMBERS", keys[i])); err != nil {
				return err
			}
			fmt.Fprintln(writer, keys[i], dataSlice)
		}
	}
	fmt.Println("Store Data Finish")
	return nil
}
