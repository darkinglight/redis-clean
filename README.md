[简体中文](README.md) ∙ [English](README-en.md)

# redis清理工具
> 使用正则表达式查询清理redis key。
> 本工具尤其适用于redis key数量>千万，CGI工具查询较慢的情况。 
> 也可以用来批量保存数据到本地。 

## 配置
默认加载的配置文件名为config.yaml, 可以通过-config参数执行你自己的配置文件  
配置文件默认的加载路径如下：  
1. -config参数的绝对路径
2. 当前工作目录的相对路径
3. 可执行文件目录的相对路径

执行程序前需要更新的配置：
1. 替换默认的redis配置替换为目标redis配置，slave从库配置可以不配置，但会增加主库的查询压力
2. 更新需要查找的key的正则表达式，如：需要查找testA,testB,testC,则配置test*
3. 如果需要可以更改iterNum的值,控制单次查询redis遍历的key的数量，值越大，单次查询阻塞越久，默认值10000
4. 如果需要可以更改deleteNum的值，控制单次删除key的数量，值越大，单次阻塞越久，默认值100

## 安装
在根目录执行`make`，会生成bin目录，并在bin目录生成linux可执行文件redis-clean和windows可执行文件redis-clean.exe 

## 使用
redis-clean [-config "path/to/configfile.yaml"] [-enableDeleteData] [-enableSaveData] [-enableSaveKey]
主要流程如下：
1. 查找redis key：根据配置的正则表达式，使用redis命令`scan iter match pattern count iterNum`进行key的查找
2. 保存匹配的key: -enableSaveKey,保存所有匹配的key到key.txt
3. 保存redis数据：-enableSaveData，保存匹配的key的数据到当前目录的data.txt文件
4. 删除redis key: -enableDeleteData,删除匹配的key

实际使用界面如下：
```
$ ./redis-clean -enableDeleteData
connect to redis master suucess.
connect to redis slave suucess.
[Search Key By Pattern TASK:ONCE:1*] Process: 2%
Search Key Finish. Total Key Number: 135219134, Searched Key Number: 3020000, Match Key Number: 1002680
[Delete Keys] Process: 100%
Delete Keys Finish. Match Size:1002680; Delete Size:1002680
[Search Key By Pattern TASK:ONCE:1*] Process: 4%
Search Key Finish. Total Key Number: 134216454, Searched Key Number: 6020000, Match Key Number: 2003671
[Delete Keys] Process: 100%
Delete Keys Finish. Match Size:1000991; Delete Size:1000991
[Search Key By Pattern TASK:ONCE:1*] Process: 6%
Search Key Finish. Total Key Number: 133215466, Searched Key Number: 9020000, Match Key Number: 3004421
[Delete Keys] Process: 100%
Delete Keys Finish. Match Size:1000750; Delete Size:1000750
[Search Key By Pattern TASK:ONCE:1*] Process: 9%
Search Key Finish. Total Key Number: 132214719, Searched Key Number: 12020000, Match Key Number: 4006669
[Delete Keys] Process: 100%
Delete Keys Finish. Match Size:1002248; Delete Size:1002248
[Search Key By Pattern TASK:ONCE:1*] Process: 10%   
```

## 帮助
redis-clean -h

## 测试
> 使用命令`seq 200000 | awk '{print "test"$1}' | xargs -n 10000 redis-cli -h localhost -p 6379 mset`添加测试数据  
> 执行本脚本进行测试
