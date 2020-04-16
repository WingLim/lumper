## Overview

lumper 是一个简单轻量的容器管理引擎。lumper 可以让你将应用程序和系统基础架构分离，以便快速部署服务和交付软件。通过 lumper 可以让你的开发环境和生产环境同步。并且因为轻量，可以无痛部署到物联网边缘端，且占用体积小。



## lumper 架构

lumper 是一个 CLI 工具，通过命令行获取用户输入，再对底层进行操作，从而进行容器的创建和管理。

![简易架构图](D:\dev\lumper\简易架构图.png)

### lumper 对象

#### 镜像

镜像是一个只读的模板，给予应用程序一个运行环境。lumper 可以基于 Docker 中导出的镜像来运行，同时也支持`lumper commit` 操作来打包自己的镜像。

#### 容器

容器是镜像的可运行实例，通过 lumper 你可以创建，停止，删除一个容器，还可以将宿主机的文件/文件夹挂载到容器内。

容器内默认不存在持久性储存，如果你将一个容器删除，你在容器内储存的数据以及改动都会被删除，但你可以通过挂载宿主机文件来保持数据的持久性。

你可以通过 `lumper run -t busybox sh`来创建和运行一个容器。

### 底层技术

lumper 是用 Go 语言来编写的，通过 Linux 内核的一些特性，来实现容器技术。

#### Namespaces

lumper 使用 namespaces 来为容器提供与宿主机隔离的工作空间，在创建一个容器时，lumper 会为这个容器创建一组 namespaces。

使用的 namespaces 如下：

- PID Namespace：进程隔离
- Mount Namespace：管理文件系统挂载点
- UTS Namespace：隔离内核和版本标识符
- IPC Namespace：管理进程间通信
- Network Namespace：管理网络接口

#### Control groups

lumper 还使用了 Linux 的 Cgroups 技术，通过 Cgroups，可以对一组进程及它的子进程进行资源限制、控制和通知，这些资源包括CPU、内存、网络等。例如：lumper 可以通过 Cgroups 来限制一个容器运行最多能占用的运行内存。

#### Union file systems

lumper 使用 Union file systems 来对镜像及容器内文件进行分层存储，而且使用了 **overlay** 作为存储驱动，相比 AUFS 效率更高。

## Lumper CLI

### lumper commit

#### Description

将容器的更改重新打包成一个新的镜像

#### Usage

```bash
lumper commit CONTAINER IMAGENAME
```

#### Extended description

它能够将在容器中的文件修改或设置重新打包成一个新的镜像，可以用于创建一个新的容器，减少了配置容器的工作步骤，提高效率。

容器打包时不会将数据卷中的数据一起打包。

#### Examples

```bash
$ lumper commit container1 image1
```



### lumper exec

#### Description

在正在运行的容器中执行一个命令

#### Usage

```bash
lumper exec CONTAINER COMMAND [ARG...]
```



#### Examples

首先，启动一个容器

```bash
$ lumper run -d --name container1 busybox top
```

然后在容器内执行一个命令

```bash
$ lumper exec container1 sh
```



### lumper list

#### Description

列出所有容器

#### Usage

```bash
lumper list
```



#### Example

```bash
$ lumper list
ID             NAME         PID         STATUS      COMMAND     CREATED
296885565726   container1   8604        running     sh          2020/3/4 12:53:04
805568624921   container2   8621        running     sh          2020/3/4 12:53:27
```



### lumper logs

#### Description

获取容器的日志

#### Usage

```bash
lumper logs CONTAINER
```



#### Example

```bash
$ lumper logs container2
{"level":"info","msg":"initing","time":"2020-03-04T12:53:27+08:00"}
{"level":"info","msg":"current location is /var/lib/lumper/overlay2/container2/merged","time":"2020-03-04T12:53:27+08:00"}
{"level":"info","msg":"find path /bin/sh","time":"2020-03-04T12:53:27+08:00"}
```



### lumper rm

#### Description

删除一个容器

#### Usage

```bash
lumper rm CONTAINER
```

#### Extended description

当容器状态为运行中时，容器无法被删除，需要先停止容器再进行删除。

#### Example

```bash
$ lumper rm container2
```



### lumper run

#### Description

创建并运行一个容器

#### Usage

```bash
lumper run [OPTIONS] IMAGE [COMMAND] [ARG...]
```

#### Options

| Name, shorthand   | Description          |
| ----------------- | -------------------- |
| `--cpuset`        | 允许执行的 CPU       |
| `--cpushare , -c` | CPU 共享（相对权重） |
| `--detach , -d`   | 在后台运行容器       |
| `--env , -e`      | 设置环境变量         |
| `--memory , -m`   | 内存限制             |
| `--name`          | 为容器指定名称       |
| `--tty , -t`      | 给容器分配伪 tty     |
| `--volume , -v`   | 绑定数据卷           |

#### Examples

在后台运行一个名为 `container1` ，使用 `busybox` 作为底层镜像，并执行 `sh` 命令的容器

```bash
$ lumper run -d --name container1 busybox sh
```

运行一个名为 `container2` ，使用 `busybox` 作为底层镜像，执行 `sh` 并分配伪tty 的容器

```bash
$ lumper run -t --name container2 busybox sh
```



### lumper stop

#### Description

停止一个容器

#### Usage

```bash
lumper stop CONTAINER
```

#### Examples

```bash
$ lumper stop container1
```









