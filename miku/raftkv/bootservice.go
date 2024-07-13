// Package raftkv Package raft 持久化层的raft包装实现，引用持久层模块
package raftkv

import (
	"esdeath_go/miku/base"
	"esdeath_go/miku/persistent"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"net"
	"os"
	"path/filepath"
	"time"
)

// raft 相关常量
const (
	raftStoreDir = "raft_data"
	// 用于创建 SnapshotStore 的参数，控制保留多少快照, 必须至少为1。
	defRaftRetainSnapshotCount = 2
	raftTimeout                = 10 * time.Second
	raftLogFile                = "raft-log.bolt"
	raftStableFile             = "raft-stable.bolt"
)

// command 枚举
const (
	Insert = "insert"
	Delete = "delete"
)

var log = base.BuildLogger("raft", hclog.Debug, os.Stdout)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Error("get pwd err: ", err)
		os.Exit(0)
	}
	p := filepath.Join(cwd, raftStoreDir)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err := os.MkdirAll(raftStoreDir, 0700); err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
	}
	log.Info("raft store path", "path", p)
}

// Start 启动一个raft包装的存储实例
//
// raftNodeId 	raft节点的ID，非必填，如果为空，则等于raftNodeAddr
//
// raftNodeAddr raft节点的地址，必填
//
// joinAddr 	非必填，如果为空，则表示当前节点是集群的第一个节点，否则表示当前节点要加入的集群地址
func Start(raftNodeId, raftNodeAddr, joinAddr string) (*RaftKV, error) {
	if raftNodeId == "" {
		raftNodeId = raftNodeAddr
	}

	pr := newRaftKV()
	pr.raftBind = raftNodeAddr

	if err := pr.open(raftNodeId, joinAddr == ""); err != nil {
		return nil, err
	}
	return pr, nil
}

// RaftKV 使用raft和kv引擎实现的一个分布式kv存储
type RaftKV struct {
	raftBind      string
	raft          *raft.Raft
	ExistingState bool
	persistent.DbPersist
	l hclog.Logger
}

// newRaftKV returns a new RaftKV.
func newRaftKV() *RaftKV {
	return &RaftKV{
		DbPersist: persistent.PersistInstance,
		l:         log,
	}
}

// open 开启 raft节点，用于raft 内部集群之间相互通讯传递消息
func (r *RaftKV) open(localID string, isBoot bool) error {
	// raft config
	raftConfig := raft.DefaultConfig()
	raftConfig.Logger = log
	raftConfig.LocalID = raft.ServerID(localID)
	raftConfig.SnapshotInterval = raftConfig.SnapshotInterval * 5
	raftConfig.SnapshotThreshold = raftConfig.SnapshotThreshold * 5

	raftConfig.HeartbeatTimeout = raftConfig.HeartbeatTimeout * 5
	raftConfig.ElectionTimeout = raftConfig.ElectionTimeout * 5
	raftConfig.CommitTimeout = raftConfig.CommitTimeout * 5
	raftConfig.LeaderLeaseTimeout = raftConfig.LeaderLeaseTimeout * 5

	logStore, err := raftboltdb.New(raftboltdb.Options{
		Path: filepath.Join(raftStoreDir, raftLogFile),
	})

	stableStore, err := raftboltdb.New(raftboltdb.Options{
		Path: filepath.Join(raftStoreDir, raftStableFile),
	})

	rst := defRaftRetainSnapshotCount
	if base.MkConfig.RaftRetainSnapshotCount > 0 {
		rst = base.MkConfig.RaftRetainSnapshotCount
	}
	snapshots, err := raft.NewFileSnapshotStore(raftStoreDir, rst, os.Stderr)
	if err != nil {
		return fmt.Errorf("file snapshot service: %s", err)
	}

	// Transport 是raft集群内部节点之间的通信渠道，节点之间需要通过这个通道来进行日志同步、leader选举等。
	addr, err := net.ResolveTCPAddr("tcp", r.raftBind)
	if err != nil {
		return err
	}
	transport, err := raft.NewTCPTransport(addr.String(), addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	ra, err := raft.NewRaft(raftConfig,
		(*fsm)(r),
		logStore,
		stableStore,
		snapshots,
		transport)
	if err != nil {
		return fmt.Errorf("new raft: %s", err)
	}
	r.raft = ra

	hasState, err := raft.HasExistingState(logStore, stableStore, snapshots)
	if err != nil {
		log.Error("failed to retrieve raft state", "err", err)
		return err
	}
	r.ExistingState = hasState
	log.Info("hasState", hasState)

	if !r.ExistingState && isBoot {
		log.Info("bootstrapping new cluster", "localID", raftConfig.LocalID, "addr", transport.LocalAddr())
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raftConfig.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		// 第一个节点通过 bootstrap 的方式启动，它启动后成为领导者
		ra.BootstrapCluster(configuration)
	}

	return nil
}

// join 处理节点接入集群的请求处理
func (r *RaftKV) join(nodeID, addr string) error {
	r.l.Info("received join request", "nodeID", nodeID, "addr", addr)

	configFuture := r.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		r.l.Error("failed to get raft configuration", "err", err)
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// 如果节点已经存在，并且具有要加入的节点的 ID 或 地址，则可能需要首先从配置中删除该节点。
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// 但是，如果ID和地址都相同，则不需要任何操作，甚至不需要连接操作。
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				r.l.Info("node already member of cluster, ignoring join request", "nodeID", nodeID, "addr", addr)
				return nil
			}

			future := r.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	f := r.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	r.l.Info("node joined successfully", "nodeID", nodeID, "addr", addr)
	return nil
}

// Query 获取一个key的值
func (r *RaftKV) Query(key string) (string, error) {
	one, err := r.GetOne([]byte(key))
	if err != nil {
		return "", err
	}
	return string(one), nil
}

// 注意对于大多数应用来说，只有leader能接受update(包括delete和insert)操作，
// 对于一致性不太强的，可以允许follower接受read操作,但是对于消息队列的场景来说，不存在存粹的read操作
// 无论是拉取消息还是新增消息，都是update操作，所以这里不做区分

// Insert
func (r *RaftKV) Insert(kvps []*persistent.KVPair) error {
	return r.change(&persistent.Command{Op: Insert, Kvps: kvps})
}

// Delete deletes the given key.
func (r *RaftKV) Delete(kvps []*persistent.KVPair) error {
	return r.change(&persistent.Command{Op: Delete, Kvps: kvps})
}

func (r *RaftKV) IsLeader() bool {
	return r.raft.State() == raft.Leader
}

func (r *RaftKV) RaftShutdown() error {
	return r.raft.Shutdown().Error()
}

func (r *RaftKV) change(c *persistent.Command) error {
	b, err := c.Encode()
	if err != nil {
		return err
	}
	f := r.raft.Apply(b, raftTimeout)
	return f.Error()
}
