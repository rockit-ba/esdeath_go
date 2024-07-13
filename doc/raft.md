https://thesecretlivesofdata.com/raft/

假设我们有一个单节点的一个存储单个值的数据库服务器，有一个服务端添加kv和查询kv的客户端。这个时候通过一个节点就该价值达成一致或共识很容易，因为它根本就不需要达成共识。

但是如果但是如果我们有多个节点，我们如何达成共识？这里的共识是指：如果一个节点添加了一个kv，其他节点也要添加这个kv，如果一个节点查询了一个kv，其他节点也要查询这个kv。

raft就是为了解决共识问题的。

# 概述

## 领导选举

raft 的集群中的node 可能存在三种状态之一：

follower（跟随者），leader（领导），candidate（候选者）

1 集群开始所有的节点都是follower状态。

2 如果有的follower在一个随机的超时时间内没有收到leader的心跳，则会进入candidate状态。

3 candidate状态的node会开始向其它node发送请求投票。

4 candidate获得了集群中大多数选票，则candidate转换为leader状态。

这个过程称之为领导选举（*Leader Election*）。

当选举出leader之后，后面集群内所有的变更操作将都由leader接收处理。



## 日志复制

1 首先客户端将set(8)变更请求发送到leader

2 leader 会向本地写入一条 log entry。此时log entry 处于未提交状态，也就是说这条log entry还没有被集群中的多数节点 写入。因此未提交状态的 变更操作不会应用到应用层（对于我们的kv服务来说,还未将8 set到我们的kv系统）。

3 要提交这条  log entry，首先需要将其复制给其它的follower node。

4 然后leader等待复制请求的响应，当集群中的多数 follower node写入复制过去的这条  log entry之后，这条 log entry变为已提交状态，然后会被应用到应用层。（对于我们的kv服务来说，已经将 8 set到了服务中）

5 然后leader会通知follower 这条 log entry已经被成功提交，然后follower node 也会将这条日志提交，也就是应用到它们的应用层。

到此为止，集群中的所有状态已经完全一致。我们称之为这个集群内部达成了共识。



# leader选举

在raft选举机制中，有两个超时控制变量。

首先是选举超时（election timeout）。选举超时是node从follower状态变为等待变为candidate 状态的时间。也就是上面提到的接收leader心跳的超时时间。选举超时时间随机设置在 150 毫秒到 300 毫秒之间。注意随机是为了降低多个节点同时发起投票的概率。

当选举超时之后，follower节点进入candidate状态。

先自增当前节点的任期号（Term:+1），

并且给自己投一票（Vote Count + 1），

然后并发向其它节点发起投票选举，请求投票给自己。

如果接收投票的follower在本任期（Term）内尚未投票，那么它将投票给candidate。将自己的任期号设置为candidate的任期号，Voted For 设置为 candidate node ，并且重置自己的选举超时时间。

注意：若follower接收到较高任期值的投票请求（而不是同等或者更低的Term），follower会更新其Term并重置投票状态，以此来保证Log一致性。

获得多数投票之后，candidate 转变为leader状态。

一旦成为leader，leader就会开始向集群中的follower发送 append entries 消息。这些消息按照心跳超时（heartbeat timeout）指定的时间间隔发送。follower接收到消息之后会重置自己的选举超时时间，然后响应。

这个选举期限（Term）将持续到follower 停止接收心跳并成为candidate为止。

### election timeout 和 heartbeat timeout

注意election timeout 和 heartbeat timeout在某些场景下的作用是重叠的，这里再详细总结一下：

**Election timeout**（选举超时）是指在Raft集群中，一个节点**等待成为领导者**（leader）或**接收来自当前领导者的心跳消息的最大时间**。如果一个节点在选举超时时间内没有收到来自领导者的心跳消息或其他节点的投票请求，它会认为领导者已经失效，开始发起新的选举。

**具体细节**：

- **随机时间间隔**：每个节点的选举超时通常是一个随机值，这样可以防止多个节点同时发起选举，避免选举冲突（split vote）。
- **超时范围**：选举超时的值通常在150ms到300ms之间，但具体值可以根据网络条件和应用需求进行调整。
- **重新选举**：如果在选举超时时间内，节点没有收到大多数选票，它会增加选举超时时间并重新发起选举。

**Heartbeat timeout**（心跳超时）是指领导者节点定期向所有跟随者节点发送心跳消息的时间间隔。心跳消息的作用是告知跟随者节点当前领导者仍然存活，防止跟随者节点发起选举。

**具体细节**：

- **固定时间间隔**：心跳超时时间通常设置为一个固定值，通常会设置为选举超时的1/3到1/10之间，例如50ms到100ms。这个值应确保频繁到可以在节点失效时及时发起新的选举。
- **领导者责任**：领导者节点必须在心跳超时内向所有跟随者节点发送心跳消息。如果跟随者节点在心跳超时内没有收到心跳消息，它们会认为领导者失效，并进入候选状态发起选举。



## leader停止之后集群重选流程

1 leader 在Heartbeat timeout时间内没有向follower发送心跳，此时follower中有一个node 会先转变为 candidate，自增自己的任期号，给自己投一票，然后向集群中的其它node 发起投票。如果集群中多数节点统一投票，则此candidate节点变为leader。

重点是需要多数票才能保证每个任期只能选举一名领导人。所谓的多数票数就是 n/2 +1个票数，n代表集群的node数量。因此，如果集群的n为3，那么当其中两台node都stop的情况下，集群是无法选举出新的leader的，也就是说，超过多数的节点stop之后，raft集群就是不可用的。

## 投票分裂

如果碰巧超过一个的follower变成了candidate，这时会发生投票分裂。

比如4个node集群，nodeA和nodeB 发起了同任期号的投票。它们各自给自己投了一票，然后向集群的其它node发起了投票。

nodeA和nodeB都有 2 票，并且在本任期内不能再获得更多选票。节点将等待新的选举并重试。由于随机的选举超时时间，最终会只有一个node发起投票，并最终成为leader。



# 日志复制

一旦选出了领导者，我们就需要将由leader接收的所有更改复制到所有node。日日志复制和leader选举同样重要。

日志复制也是通过 append entries 消息来完成的，没错，和上面的心跳消息一样。不同的是心跳消息是一条空的 append entries 消息，而日志复制的append entries 是带有log entry 的非空消息。

流程：

1 客户端向领导者发送更改。

2 更改操作首先会被 append 到 leader的log 中。此时这条日志处于未提交状态。

3 然后这条更改日志会在下一个心跳时通过append entries 消息发送给集群中的followers。

4 一旦leader确认集群中的多数follower 写入了自己的log之后，就会将这条日志提交，提交意味着应用到raft上层的应用层。

5 一旦leader提交了客户端的此次更改之后，就会响应客户端此次更改的结果。

6 follower在随后的日志同步中会将自己的这条log 也进行提交。

## 网络分区一致性

Raft 甚至可以在网络分区时保持一致性。主要会用到Term.

如果我们原来有一个五个节点的集群 A B C D E ，集群的Term为1,leader 为B。

现在这个集群产生了网络分区,  A B可以通信，C D E之间可以通信。此时原来的集群被分区了。

C D E 被分区了，C节点第一个到了选举超时，成为了candidate，自增自己的Term 为2，投自己一票，然后向 D E发起投票。然后C节点成为了 C D E集群中的leader。

假设我们现在有两个客户端。

一个客户端将尝试将节点 B 的值设置为“3”。因为此时节点B无法完成多数节点写入log（此时只有A B两个节点，最多两个节点，而原来的集群有五个节点，多数节点是 3），因此，这条日志在B 节点上一直未提交。

另一个客户端将尝试将节点 C 的值设置为“8”。

这个操作会成功，因为C D E三个节点会复制日志，达到了多数节点的条件，这个操作在 C被提交。

这个时候我们再恢复网络分区，会发生什么呢？

网络恢复时，任何follower接受到不同步的日志序列时，一定会要求leader通过AppendEntries消息中的prevLogIndex和prevLogTerm字段找到一致点并“补偿”日志来恢复一致性。

注意此时 B leader的Term是1，C leader的Term是2，因此，B leader会退居为follower。

这个时候节点 A 和 B 都会回滚其未提交的log  entry，并写入新领导者C 的日志。此时，整个集群的日志又一致了。