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
func findKeys(conn redis.Conn, pattern string, iterNum int, keyChannel chan<- string) {
	//get total key count
	totalCount, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
        close(keyChannel)
		return
	}

	var searchProcess = NewProcess(fmt.Sprintf("Search Key By Pattern %s", pattern), totalCount)

	var matchNum int
    iter := 0
    var keys []string
    for round := 1; ; round++ {
		if arr, err := redis.MultiBulk(conn.Do("SCAN", iter, "MATCH", pattern, "COUNT", iterNum)); err != nil {
            close(keyChannel)
            fmt.Println(err)
            return
		} else {
			iter, _ = redis.Int(arr[0], nil)
			keys, _ = redis.Strings(arr[1], nil)
		}
		if len(keys) > 0 {
            matchNum += len(keys)
            for _, key := range keys {
                keyChannel <- key
            }
		}
		searchProcess.Print(round * iterNum)
		if iter == 0 {
			break
		}
	}
    fmt.Printf("Search Key Finish. Total Key Number: %d, Match Key Number: %d\n", totalCount, matchNum)

    close(keyChannel)
	return
}

/**
 * 删除redis key
 */
func deleteKeys(conn redis.Conn, keys []string, nums int) error {
	size := len(keys)
	var totalDeleteNum int
	var part []string
	var deleteProcess = NewProcess("Delete Keys", size)
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
		deleteProcess.Print(totalDeleteNum)
	}
	fmt.Printf("Delete Keys Finish. Match Size:%d; Delete Size:%d\n", size, totalDeleteNum)
	return nil
}

/**
 * 保存redis数据
 */
func storeData(conn redis.Conn, keys []string, filePath string, typeNum int, dataNum int) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	//get type of all keys
	stringKeys, listKeys, zsetKeys, hashKeys, setKeys, err := getType(conn, keys, typeNum)
	if err != nil {
		return err
	}

	//fetch data by pipeline
	if err = fetchString(conn, stringKeys, writer, dataNum); err != nil {
		return err
	}
	if err = fetchList(conn, listKeys, writer, dataNum); err != nil {
		return err
	}
	if err = fetchZset(conn, zsetKeys, writer, dataNum); err != nil {
		return err
	}
	if err = fetchHash(conn, hashKeys, writer, dataNum); err != nil {
		return err
	}
	if err = fetchSet(conn, setKeys, writer, dataNum); err != nil {
		return err
	}

	return nil
}

func getType(conn redis.Conn, keys []string, num int) ([]string, []string, []string, []string, []string, error) {
	var dataNum int = 0
	var stringKeys, listKeys, zsetKeys, hashKeys, setKeys []string
	size := len(keys)
	var typeProcess = NewProcess("Store Data Get Key Type", size)
	for i := 0; i < size; i++ {
		typeProcess.Print(i + 1)
		conn.Send("TYPE", keys[i])
		dataNum++
		if dataNum%num == 0 || i == size-1 {
			conn.Flush()
			for j := 0; j < dataNum; j++ {
				keyType, err := redis.String(conn.Receive())
				if err != nil {
					return nil, nil, nil, nil, nil, err
				}
				switch keyType {
				case "string":
					stringKeys = append(stringKeys, keys[i-dataNum+j+1])
				case "list":
					listKeys = append(listKeys, keys[i-dataNum+j+1])
				case "zset":
					zsetKeys = append(zsetKeys, keys[i-dataNum+j+1])
				case "hash":
					hashKeys = append(hashKeys, keys[i-dataNum+j+1])
				case "set":
					setKeys = append(setKeys, keys[i-dataNum+j+1])
				}
			}
			dataNum = 0
		}
	}
	return stringKeys, listKeys, zsetKeys, hashKeys, setKeys, nil
}

func fetchString(conn redis.Conn, keys []string, writer *bufio.Writer, num int) error {
	keysLen := len(keys)
	currentNum := 0
	var stringProcess = NewProcess("Store String Data", keysLen)
	for i, key := range keys {
		stringProcess.Print(i + 1)
		conn.Send("GET", key)
		currentNum++
		if currentNum%num == 0 || i == keysLen-1 {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.String(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i-currentNum+j+1], data)
			}
			currentNum = 0
		}
	}
	return nil
}

func fetchList(conn redis.Conn, keys []string, writer *bufio.Writer, num int) error {
	keysLen := len(keys)
	currentNum := 0
	var listProcess = NewProcess("Store List Data", keysLen)
	for i, key := range keys {
		listProcess.Print(i + 1)
		conn.Send("LRANGE", key, 0, -1)
		currentNum++
		if currentNum%num == 0 || i == keysLen-1 {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.Strings(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i-currentNum+j+1], data)
			}
			currentNum = 0
		}
	}
	return nil
}

func fetchZset(conn redis.Conn, keys []string, writer *bufio.Writer, num int) error {
	keysLen := len(keys)
	currentNum := 0
	var zsetProcess = NewProcess("Store Zset Data", keysLen)
	for i, key := range keys {
		zsetProcess.Print(i + 1)
		conn.Send("ZRANGE", key, 0, -1, "WITHSCORES")
		currentNum++
		if currentNum%num == 0 || i == keysLen-1 {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.StringMap(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i-currentNum+j+1], data)
			}
			currentNum = 0
		}
	}
	return nil
}

func fetchHash(conn redis.Conn, keys []string, writer *bufio.Writer, num int) error {
	keysLen := len(keys)
	currentNum := 0
	var hashProcess = NewProcess("Store Hash Data", keysLen)
	for i, key := range keys {
		hashProcess.Print(i + 1)
		conn.Send("HGETALL", key)
		currentNum++
		if currentNum%num == 0 || i == keysLen-1 {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.StringMap(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i-currentNum+j+1], data)
			}
			currentNum = 0
		}
	}
	return nil
}

func fetchSet(conn redis.Conn, keys []string, writer *bufio.Writer, num int) error {
	keysLen := len(keys)
	currentNum := 0
	var setProcess = NewProcess("Store Set Data", keysLen)
	for i, key := range keys {
		setProcess.Print(i + 1)
		conn.Send("SMEMBERS", key)
		currentNum++
		if currentNum%num == 0 || i == keysLen-1 {
			conn.Flush()
			for j := 0; j < currentNum; j++ {
				data, err := redis.Strings(conn.Receive())
				if err != nil {
					return err
				}
				fmt.Fprintln(writer, keys[i-currentNum+j+1], data)
			}
			currentNum = 0
		}
	}
	return nil
}
