package main

import (
	"bufio"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"os"
    "sync"
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
func findKeys(conn redis.Conn, pattern string, iterNum int, keyChannel chan<- string, p *process, wg *sync.WaitGroup) {
    defer wg.Done()
	//get total key count
	totalCount, err := redis.Int(conn.Do("DBSIZE"))
	if err != nil {
        close(keyChannel)
		return
    } else {
        p.SetTotal(totalCount)
    }

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
            p.IncrMatchNum(len(keys))
            for _, key := range keys {
                keyChannel <- key
            }
		}
        p.IncrSearchNum(iterNum)
		if iter == 0 {
			break
		}
	}

    close(keyChannel)
}

/**
 * 删除redis key
 */
func deleteKeys(conn redis.Conn, keys <-chan string, nums int, p *process, wg *sync.WaitGroup) {
    defer wg.Done()
	var argNum int
    args := redis.Args{}
    for key := range keys {
		args = args.Add(key)
        argNum++
        if argNum == nums {
		    deleteNum, err := redis.Int(conn.Do("DEL", args...))
		    if err != nil {
                fmt.Println(err)
                return
		    }
            args = redis.Args{}
            argNum = 0
            p.IncrDeleteNum(deleteNum)
        }
	}
    deleteNum, err := redis.Int(conn.Do("DEL", args...))
    if err != nil {
        fmt.Println(err)
    } else {
        p.IncrDeleteNum(deleteNum)
    }
}

/**
 * 保存redis数据
 */
func storeData(conn redis.Conn, keyChan <-chan string, filePath string, typeNum int, dataNum int) {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
        fmt.Println(err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

    var keys []string
    for key := range keyChan {
        keys = append(keys, key)
        if len(keys) == dataNum {
            err = storeDataOnce(conn, keys, writer, typeNum, dataNum)
            if err != nil {
                fmt.Println(err)
                return
            }
            keys = []string{}
        }
    }
    if len(keys) > 0 {
        err = storeDataOnce(conn, keys, writer, typeNum, dataNum)
        if err != nil {
            fmt.Println(err)
        }
    }
}

func storeDataOnce(conn redis.Conn, keys []string, writer *bufio.Writer, typeNum int, dataNum int) error{
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
	for i := 0; i < size; i++ {
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
	for i, key := range keys {
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
	for i, key := range keys {
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
	for i, key := range keys {
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
	for i, key := range keys {
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
	for i, key := range keys {
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
