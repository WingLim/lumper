# lumper
A simple container runntime implementation

## Todo
- 容器
  - 操作
    - [x] 创建
      - [x] 容器名
      - [x] 后台运行
      - [x] 环境变量
    - [x] 资源限制
      - [x] 内存
      - [x] cpushare
      - [x] cpuset
    - [x] 停止
    - [x] 进入
  - [x] 管理
    - [x] 查看
    - [x] 删除
  - [x] 日志
  - [ ] 网络
  - [ ] 创建网络
    - [ ] 连接网络
    - [ ] 查看网络
    - [ ] 地址分配
  
- 镜像
  - [x] AUFS
    - [ ] overlay2
  - [x] volume 数据卷
  - [x] 打包
    - [x] commit
    - [x] 单独隔离的文件系统
