## [1.0.0] - <2020/05/15>
### 进度展示优化
* 通过context收集数据，定时刷新
### 数据过大内存溢出优化
* 通过channel逐步处理，后续流程未处理完成，前置流程阻塞等待