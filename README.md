# redis-clean tool

## Build
`make`
this cammand will generate two excutable binary file in bin directory

## Config
put the config.yaml on where you execute this tool or you can load config by -config. config demo is config.yaml.
1. relace the redis connect config with your host and so on.
2. change the keys to your pattern, like test* can search testa, testb, test:set and so on.
3. change iterNum if needed, this is number in redis command `scan iterator match pattern count iterNum`
4. chagne deleteNum if needed, this is number of elements in redis command `del key1 key2 key3 ...`

## Usage
redis-clean [-config "path/to/configfile.yaml"]

## Help
redis-clean -h
