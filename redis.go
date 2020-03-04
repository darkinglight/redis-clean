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

	//get type of all keys
	var storeProcess = process{"Store Data Get Key Type", 0}
	var dataNum int = 0
	var keyType string
	var stringKeys, listKeys, zsetKeys, hashKeys, setKeys []string
	size := len(keys)
	for i := 0; i < size; i++ {
		storeProcess.Print((i + 1) * 100 / size)
		conn.Send("TYPE", keys[i])
		dataNum++

		if dataNum%100 == 0 || i == size-1 {
			conn.Flush()
			for j := 0; j < dataNum; j++ {
				if keyType, err = redis.String(conn.Receive()); err != nil {
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
	if err = fetchString(conn, stringKeys, writer); err != nil {
		return err
	}
	if err = fetchList(conn, listKeys, writer); err != nil {
		return err
	}
	if err = fetchZset(conn, zsetKeys, writer); err != nil {
		return err
	}
	if err = fetchHash(conn, hashKeys, writer); err != nil {
		return err
	}
	if err = fetchSet(conn, setKeys, writer); err != nil {
		return err
	}

	fmt.Println("Store Data Finish")
	return nil
}

func fetchString(conn redis.Conn, keys []string, writer *bufio.Writer) error {
	var stringProcess = process{"Store String Data", 0}
	keysLen := len(keys)
	currentNum := 0
	for i, key := range keys {
		stringProcess.Print((i + 1) * 100 / keysLen)
		conn.Send("GET", key)
		currentNum++
		if currentNum%100 == 0 || i-1 == keysLen {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.String(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i], data)
			}
			currentNum = 0
		}
	}
	return nil
}

func fetchList(conn redis.Conn, keys []string, writer *bufio.Writer) error {
	var listProcess = process{"Store List Data", 0}
	keysLen := len(keys)
	currentNum := 0
	for i, key := range keys {
		listProcess.Print((i + 1) * 100 / keysLen)
		conn.Send("LRANGE", key, 0, -1)
		currentNum++
		if currentNum%100 == 0 || i-1 == keysLen {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.Strings(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i], data)
			}
			currentNum = 0
		}
	}
	return nil
}

func fetchZset(conn redis.Conn, keys []string, writer *bufio.Writer) error {
	var zsetProcess = process{"Store Zset Data", 0}
	keysLen := len(keys)
	currentNum := 0
	for i, key := range keys {
		zsetProcess.Print((i + 1) * 100 / keysLen)
		conn.Send("ZRANGE", key, 0, -1, "WITHSCORES")
		currentNum++
		if currentNum%100 == 0 || i-1 == keysLen {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.StringMap(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i], data)
			}
			currentNum = 0
		}
	}
	return nil
}

func fetchHash(conn redis.Conn, keys []string, writer *bufio.Writer) error {
	var hashProcess = process{"Store Hash Data", 0}
	keysLen := len(keys)
	currentNum := 0
	for i, key := range keys {
		hashProcess.Print((i + 1) * 100 / keysLen)
		conn.Send("HGETALL", key)
		currentNum++
		if currentNum%100 == 0 || i-1 == keysLen {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.StringMap(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i], data)
			}
			currentNum = 0
		}
	}
	return nil
}

func fetchSet(conn redis.Conn, keys []string, writer *bufio.Writer) error {
	var setProcess = process{"Store Set Data", 0}
	keysLen := len(keys)
	currentNum := 0
	for i, key := range keys {
		setProcess.Print((i + 1) * 100 / keysLen)
		conn.Send("SMEMBERS", key)
		currentNum++
		if currentNum%100 == 0 || i-1 == keysLen {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.Strings(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i], data)
			}
			currentNum = 0
		}
	}
	return nil
}
