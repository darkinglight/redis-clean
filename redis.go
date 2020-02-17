package main

import (
	"bufio"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"os"
)

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

	//get type of all keys
	var dataNum int = 0
	var stringKeys, listKeys, zsetKeys, hashKeys, setKeys []string
	for i := 0; i < size; i++ {
		conn.Send("TYPE", keys[i])
		dataNum++

		if dataNum%100 == 0 || i == size-1 {
			conn.Flush()
			for j := 0; j < dataNum; j++ {
				if v, err := conn.Receive(); err != nil {
					return err
				}
				if keyType, err = redis.String(v); err != nil {
					return err
				}
				switch keyType {
				case "string":
					stringKeys = append(stringKeys, keys[i])
				case "list":
					listKeys = append(listKeys, keys[i])
				case "zset":
					zsetKeys = append(zsetKeys, keys[i])
				case "hash":
					hashKeys = append(hashKeys, keys[i])
				case "set":
					setKeys = append(setKeys, keys[i])
				}
			}
			dataNum = 0
		}
	}

	//fetch data by pipeline

	fmt.Println("Store Data Finish")
	return nil
}
