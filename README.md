# 开发日志

**7月31日 第1周周五**

项目开始。搭建环境，学习Go语言语法。

**8月4日 第2周周二**

完成Project1的大部分内容，完成了除RawScan之外的其它测试点

RawScan上碰到困难，迭代器获取的key是cf_key而实际上不需要cf字段

之后将与同学讨论，借鉴他人思路，解决这个问题。

**8月5日 第2周周三**

参看资料，进一步理解了思路

server中不该直接调用底层txn，而应该调用standalone_storage中的方法

修改了自己代码的整个实现思路，注意了代码规范

完成了Project1的所有测试点，约耗时8秒

**8月7日 第2周周五**

完成Project2aa的所有测试点，耗时在毫秒级

修正了Term从1开始的bug，改变了对实现思路的错误理解

若不考虑异常处理，Step函数改为switch m.MsgType更为简明

**8月10日 第3周周一**

继续做Project2ab，完成了部分测试点

对MsgAppendResponse的统计，采用按match排序后取下中位数的方法

noop也需广播出去，之前忽略了这一点

修正了若干因理解而引起的错误

**8月11日 第3周周二**

继续做Project2ab，完成了部分测试点

调整了代码结构

修正了许多先前测试未发现的bug

**8月12日 第3周周三**

完成了Project2ab的所有测试点

修正了几处遗留问题

完成了Project2ac的所有测试点

**8月19日 第4周周三**

Project2b的实现中碰到大量报错，并修正之前的代码

考虑放弃Project2b

**8月22日 第4周周六**

放弃Project2b，项目进度截止到Project2ac

回顾Project1与Project2a，整理代码

准备收工


---


# The TinyKV Course

This is a series of projects on a key-value storage system built with the Raft consensus algorithm. These projects are inspired by the famous [MIT 6.824](http://nil.csail.mit.edu/6.824/2018/index.html) course, but aim to be closer to industry implementations. The whole course is pruned from [TiKV](https://github.com/tikv/tikv) and re-written in Go. After completing this course, you will have the knowledge to implement a horizontal scalable, high available, key-value storage service with distributed transaction support and a better understanding of TiKV implementation.

The whole project is a skeleton code for a kv server and a scheduler server at initial, and you need to finish the core logic step by step:

- Project1: build a standalone key-value server
- Project2: build a high available key-value server with Raft
- Project3: support multi Raft group and balance scheduling on top of Project2
- Project4: support distributed transaction on top of Project3

**Important note: This course is still in developing, and the document is incomplete.** Any feedback and contribution is greatly appreciated. Please see help wanted issues if you want to join in the development.

## Course

Here is a [reading list](doc/reading_list.md) for the knowledge of distributed storage system. Though not all of them are highly related with this course, it can help you construct the knowledge system in this field.

Also, you’d better read the overview design of TiKV and PD to get a general impression on what you will build:

- TiKV
  - <https://pingcap.com/blog-cn/tidb-internal-1/> (Chinese Version)
  - <https://pingcap.com/blog/2017-07-11-tidbinternal1/> (English Version)
- PD
  - <https://pingcap.com/blog-cn/tidb-internal-3/> (Chinese Version)
  - <https://pingcap.com/blog/2017-07-20-tidbinternal3/> (English Version)

### Getting started

First, please clone the repository with git to get the source code of the project.

``` bash
git clone https://github.com/pingcap-incubator/tinykv.git
```

Then make sure you have installed [go](https://golang.org/doc/install) >= 1.13 toolchains. You should also have installed `make`.
Now you can run `make` to check that everything is working as expected. You should see it runs successfully.

### Overview of the code

![overview](doc/imgs/overview.png)

Same as the architecture of TiDB + TiKV + PD that separates the storage and computation, TinyKV only focuses on the storage layer of a distributed database system. If you are also interested in SQL layer, see [TinySQL](https://github.com/pingcap-incubator/tinysql). Besides that, there is a component called TinyScheduler as a center control of the whole TinyKV cluster, which collects information from the heartbeats of TinyKV. After that, the TinyScheduler can generate some scheduling tasks and distribute them to the TinyKV instances. All of them are communicated by RPC.

The whole project is organized into the following directories:

- `kv`: implementation of the TinyKV key/value store.
- `proto`: all communication between nodes and processes uses Protocol Buffers over gRPC. This package contains the protocol definitions used by TinyKV, and generated Go code for using them.
- `raft`: implementation of the Raft distributed consensus algorithm, used in TinyKV.
- `scheduler`: implementation of the TinyScheduler which is responsible for managing TinyKV nodes and for generating timestamps.
- `log`: log utility to output log base	on level.

### Course material

Please follow the course material to learn the background knowledge and finish code step by step.

- [Project1 - StandaloneKV](doc/project1-StandaloneKV.md)
- [Project2 - RaftKV](doc/project2-RaftKV.md)
- [Project3 - MultiRaftKV](doc/project3-MultiRaftKV.md)
- [Project4 - Transaction](doc/project4-Transaction.md)

## Deploy a cluster

After you finished the whole implementation, it's runnable now. You can try TinyKV by deploying a real cluster, and interact with it through TinySQL.

### Build

```
make
```

It builds the binary of `tinykv-server` and `tinyscheduler-server` to `bin` dir.

### Run

Put the binary of `tinyscheduler-server`, `tinykv-server` and `tinysql-server` into a single dir.

Under the binary dir, run the following commands:

```
mkdir -p data
```

```
./tinyscheduler-server
```

```
./tinykv-server -path=data
```

```
./tinysql-server --store=tikv --path="127.0.0.1:2379"
```

### Play

```
mysql -u root -h 127.0.0.1 -P 4000
```
