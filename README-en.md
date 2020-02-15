[简体中文](README.md) ∙ [English](README-en.md)

# redis-clean tool
This is tool for deleting redis keys by regular expression.

## Build
`make`
this cammand will generate two excutable binary file in bin directory.

## Config
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
