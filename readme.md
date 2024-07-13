# esdeath

esdeath(艾斯德斯) 是一个专门用于解决延迟消息的分布式、强一致MQ系统。broker由go语言编写，可编译至大多数的操作系统进行运行。目前已实现java 客户端，并且集成springboot starter开箱即用。得益于grpc的兼容性，其它语言可方便地生成对应语言版本的client。

# 特点

## 支持任意时间的延迟消息

因为基于 kv 的LSMTree存储，用户可以发送超长时间的延迟消息（只要实际场景需要），延迟消息的延迟时长不会对系统产生任何性能上的影响。这一点，是相对于基于时间轮算法的延迟消息来说的，基于时间轮算法的延迟消息系统对于处理短时间的延迟消息很有效果，但是随着时间拉长，会产生更多系统资源消耗。

## 高可用集群

esdeath使用 Raft 算法保证数据副本的一致性，使用了开源稳定的 `hashicorp/raft` 作为实现。broker自集成raft 高可用，不需要第三方的一致性组件。轻松部署高可用、高一致性的broker集群。关于raft 请参考当前目录中的raft.md文件，对raft原理进行了基础的说明。

## gRPC通信

broker于client协议通过protobuf和gRPC定义，高性能，高可扩展，你可以根据broker的proto文件轻松生成指定语言版本的client。



# 设计原理

先看一下整体的架构图，其实非常简单。本身代码也不多，很精简。

![esdeath](doc\esdeath.jpg)

客户端生产消息或者消费消息首先通过gRpc生成的client和server进行通信，broker-server充当请求处理入口。请求会交由下层的raft包装的k-v数据存储。集群中的数据一致性同步通过raft层进行处理，如果leader节点不可用，则也是由这一层负责选举新的可用leader节点出来。

## LSMTree

这个是实现延迟消息的根本，如果我们不使用raft，只用此存储结构也能实现单机版的延迟消息。

简单介绍一下LSMTree，它相对于B+Tree。b+tree很多人应该是了解的，当前的事务型数据库的存储实现基本上都是基于b+tree的文件组织系统，b+tree修改数据的典型方式是in-place，即直接在原地修改数据。而LSMTree是一种基于日志的存储结构，修改数据的典型方式是append-only，也称为out-of-place，异地修改.即只追加数据，因此LSMTree中只有新增和迭代，删除和修改也是通过新增数据实现的，比如新增一个key：a, value : 1,如果想修改，不是找到这个 key a，而是直接再插入一条记录：key :a, value: 2.由于再这种存储中，数据是按照key有序的，读到key a之后就不再往下寻找，就可以读取到最新的key值。

LSMTree是实现高性能写（顺序写）数据库的关键，时序数据库和大型分布式数据库底层基本上都是基于这个模型作为存储引擎的，而不是b+tree。

这个**有序性**是我们实现延迟消息的关键。

我们的思路是：将延迟消息的时间戳作为存储的key，延迟消息会根据延迟时间错排序。

因此可以得到结论：如果我们拿到数据库中的第一条数据的延迟时间大于当前时间（未到消费时间），我们就可以认为后面也不会有达到延迟时间的消息，就可以停止迭代数据。

这是最基本的核心思路。

当然了，实际上要对key值结合MQ本身的一些业务进行更精细化的设计，目前我们的key设计是：topic+延迟时间戳+tag+唯一消息ID。为了实现RocketMQ类似的consumer group的功能，在这个key的基础上，又添加了 consumerGroup到key中。

## MQ的特点

在我实现延迟消息MQ系统的时候，我发现了一个微妙的点：MQ系统和其他存储系统不同的一个点，MQ系统没有所谓的read操作，也就是说MQ系统无法实现读写分离，因为它全是写操作。这可能有点违背我们的直觉，因为我们有时候会说“读取消息”。如果你仔细想就会明白，拉取消息不是一个读操作，而是一个写操作，因为一条消息不能被重复消息，消息消费实际上是删除操作。

因此，对于MQ系统来说，单raft集群只是为了保证可用性，而不是为了增加吞吐。

## client

消息消费使用客户端主动拉取策略，客户端只需要在死循环中调用消息拉取即可。broker端实现了类似rocketMQ的hold机制：在没有可消费的消息的时候会进行hold阻塞。

同时broker针对客户端的consumergroup实现了处理，不同consumergroup的消费者可以消费同一条消息，类似rocketMQ的consumergroup效果。

client需要处理由follower节点拒绝的请求，client如果收到节点返回 follower拒绝的状态响应，需要重新找到leader进行操作。broker提供了getRole方法用于返回当前请求节点的角色。

> 关于hold实现：
>
> 基本参考RocketMQ的Hold逻辑实现。因为我们的消费模型是pull模型，即使push模型内部也是有pull间接实现，因此这个hold是实现pull模型的关键。这个hold思路基本上是通用的，即使你在实现非MQ的系统，比如注册中心或则其它什么系统，设计pull模型的时候，这个hold思路都是可以通用的。
>
> 首先client pull消息的时候，如果broker没有可消费的消息，则hold当前请求，比如hold 20秒。但是如果中间有新消息是在20秒内要被消费的呢？那么新消息产生之后要notify这个hold，判断新消息是否满足。这是基本的思路，具体的实现可能还需要考虑更多的优化，下面是esdeath 的hold主要代码：
>
> ```go
> // 长轮询处理
> func (es *esdeathServer) holdProcess(r *pb.DelayMsgPull, delayTimestamp int64) (*PulledMsg, error) {
>    	notify := make(chan struct{}, 1)
> 	es.pullHoldNotifyContainer.Set(r.Topic, notify)
>    	holdTime := time.Duration(delayTimestamp-time.Now().UnixMilli()) * time.Millisecond
>    	log.Infoln("hold time: ", holdTime)
>    	select {
>    	case <-time.After(holdTime):
>    		es.pullHoldNotifyContainer.Pop(r.Topic)
>    		return nil, defTimeoutError
>    	case <-notify:
>    		return pullReqHandle(r)
>    	}
>    }
>    ```

# 使用

example文件夹下有 1node  2node  3node ，代表三个节点组成的集群。使用前将构建的esdeath可执行文件放入每个node文件夹中，然后分别启动三个节点。

每个node文件夹中都需要有三个文件：esdeath 可执行文件，serverConf.yaml配置文件和mikuConf.yaml配置文件。

先启动1node，看到自选举成为leader，类似： entering leader state: leader="Node at 127.0.0.1:12000 [Leader]"，然后再启动2node和3node。

下面说明一下配置文件：

serverConf.yaml主要用来设置gRpc broker的信息：

```yaml
server_addr: "127.0.0.1:50051"
```

目前只有一个配置，表示当前broker的启动服务地址，client将会连接这个地址进行通信，请注意如果是本地部署，三个节点需要使用不同的端口。

mikuConf.yaml主要用来设置raft包装的存储操作相关配置：

```yaml
# raft 创建 SnapshotStore 的参数，控制保留多少快照, 必须至少为1。
raft_retain_snapshot_count: 2
# 集群节点加入地址
join_handle_addr: "127.0.0.1:11000"
# 当前节点的raft地址
raft_node_addr: "127.0.0.1:12000"
# 当前节点的raft id，非必填，如果不填则默认为raft_node_addr
raft_node_id: "node1"
# 要加入的集群的地址，非必填，如果是新的 follower 节点则必填(对应master 的 rest_addr)
join_addr: ""
```

raft_retain_snapshot_count表示raft 保存的快照的数量，默认即可。

join_handle_addr：集群节点加入处理地址。leader启动之后需要通过此服务处理其它节点加入集群的请求。

raft_node_addr：当前raft node的服务地址，这个服务主要是raft node内部通信的服务，包括选举和日志复制。

join_addr：当前节点启动之后要加入哪个集群的服务地址，集群第一次启动需要2node和3node的join_addr为空，1node启动之后，2node和3node的join_addr填写1node的join_handle_addr。集群后面启动就不会再去使用该参数。
