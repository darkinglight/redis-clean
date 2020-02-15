[简体中文](README.md) ∙ [English](README-en.md)

# redis清理工具
> 使用正则表达式查询清理redis key。
> 本工具尤其适用于redis key数量>千万，CGI工具查询较慢的情况
> 也可以用来批量保存数据到本地

## 编译
在根目录执行`make`，会生成bin目录，并在bin目录生成linux可执行文件redis-clean和windows可执行文件redis-clean.exe 

## 配置
put the config.yaml on where you execute this tool or you can load config by -config. config demo is config.yaml.
1. relace the redis connect config with your host and so on.
2. change the keys to your pattern, like test* can search testa, testb, test:set and so on.
3. change iterNum if needed, this is number in redis command `scan iterator match pattern count iterNum`.
4. chagne deleteNum if needed, this is number of elements in redis command `del key1 key2 key3 ...`.

## Usage
redis-clean [-config "path/to/configfile.yaml"]
### find keys
this script will search redis db by `scan iter match pattern count iterNum`
### save data
you can save data to local file after search all keys if you choose 'y'
### delete keys
you can delete all searched keys after saving data if you choose 'y'

## Help
redis-clean -h

## Test
you can use this command produce test redis data   
`eq 200000 | awk '{print "test"$1}' | xargs -n 10000 redis-cli -h localhost -p 6379 mset`

## Issue
1. don't test saving data which is very big.
