package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"gopkg.in/yaml.v2"
	"io/ioutil"
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
	conf := c.getConfig(configFile)

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

	var answer string
	//save data
	fmt.Println("save data before delete keys? [y or n]")
	fmt.Scan(&answer)
	if answer == "y" {
		fmt.Println("start store data:")
		storeData(connSlave, keys, "data.txt")
	}

	//delete keys
	fmt.Println("are you sure delete these matched keys? [y or n]")
	fmt.Scan(&answer)
	if answer == "y" {
		fmt.Println("start delete keys:")
		deleteKeys(connMaster, keys, conf.DeleteNum)
	}
	fmt.Println("finished")
}

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
func (c *config) getConfig(configFile string) *config {
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		fmt.Println(err.Error())
	}
	return c
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
 * 根据正则表达式查询redis key
 */
func findKeys(conn redis.Conn, pattern string, iterNum int) ([]string, error) {
	//get total key count
	totalCount, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
		return nil, err
	}

	fmt.Printf("start search keys by pattern %s:\n", pattern)
	iter := 0
	var keysMatch, keys []string
	for i := 1; ; i++ {
		if arr, err := redis.MultiBulk(conn.Do("SCAN", iter, "MATCH", pattern, "COUNT", iterNum)); err != nil {
			return nil, err
		} else {
			iter, _ = redis.Int(arr[0], nil)
			keys, _ = redis.Strings(arr[1], nil)
		}
		fmt.Printf("process: %d%%\n", i*iterNum*100/totalCount)
		if len(keys) > 0 {
			keysMatch = append(keysMatch, keys...)
		}
		if iter == 0 {
			break
		}
	}
	fmt.Println("matched keys:", keys)
	fmt.Println("total key num:", totalCount)
	fmt.Println("matched key num:", len(keys))

	return keysMatch, nil
}

/**
 * 删除redis key
 */
func deleteKeys(conn redis.Conn, keys []string, nums int) error {
	size := len(keys)
	var part []string
	for i := 0; i*nums < size; i++ {
		if i+nums > size {
			part = keys[i:]
		} else {
			part = keys[i : i+nums]
		}
		deleteNum, err := redis.Int(conn.Do("DEL", redis.Args{}.AddFlat(part)...))
		if err != nil {
			return err
		}
		if deleteNum < size {
			fmt.Printf("not all deleted. size:%d; deleted size:%d\n", size, deleteNum)
		}
		fmt.Println("keys deleted:", part)
	}
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

	size := len(keys)
	var keyType string
	var data string
	var dataSlice []string
	var dataMap map[string]string
	for i := 0; i < size; i++ {
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
	return nil
}
