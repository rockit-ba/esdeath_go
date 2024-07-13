package miku

import (
	"bytes"
	"encoding/json"
	"errors"
	"esdeath_go/miku/base"
	"esdeath_go/miku/persistent"
	"esdeath_go/miku/raftkv"
	"fmt"
	"github.com/hashicorp/go-hclog"
	"io"
	"net/http"
	"os"
	"time"
)

var log = base.BuildLogger("raft", hclog.Info, os.Stdout)

func Start() *RMikuKv {
	raftInstance, err := raftkv.Start(base.MkConfig.RaftNodeID,
		base.MkConfig.RaftNodeAddr,
		base.MkConfig.JoinAddr)
	if err != nil {
		log.Error(fmt.Sprintf("raftkv.Start error: %s", err))
		os.Exit(1)
	}

	if err = restStart(raftInstance); err != nil {
		log.Error(fmt.Sprintf("raftkv.restStart error: %s", err))
		os.Exit(1)
	}
	return &RMikuKv{raftInstance: raftInstance}
}

const maxJoinRetry = 10

func restStart(raftInstance *raftkv.RaftKV) error {
	err := raftkv.NewJoinProcessor(base.MkConfig.JoinHandleAddr, raftInstance).Start()
	if err != nil {
		return fmt.Errorf("failed to start rest service: %s", err)
	}

	log.Info("node started", "listener", hclog.Fmt("http://%s", base.MkConfig.JoinHandleAddr))
	if raftInstance.ExistingState {
		log.Info("node already in cluster",
			"nodeID", base.MkConfig.RaftNodeID,
			"addr", base.MkConfig.RaftNodeAddr)
		return nil
	}
	if base.MkConfig.JoinAddr == "" {
		return nil
	}
	// 如果定义了joinAddr，就加入集群
	retryCount := 0
	for {
		err = joinCluster(base.MkConfig.JoinAddr, base.MkConfig.RaftNodeAddr, base.MkConfig.RaftNodeID)
		if err != nil {
			if errors.Is(err, ErrNodeJoinMsg) && retryCount < maxJoinRetry {
				retryCount++
				log.Warn(fmt.Sprintf("join cluster error: %s, retrying...", err))
				time.Sleep(time.Duration(retryCount) * time.Second)
				continue
			}
		}
		break
	}

	return err
}

var ErrNodeJoinMsg = errors.New("join cluster error")

func joinCluster(joinAddr, raftNodeAddr, nodeID string) error {
	b, err := json.Marshal(map[string]string{"addr": raftNodeAddr, "id": nodeID})
	if err != nil {
		return err
	}
	// 将当前节点信息post 提交，加入到集群中
	resp, err := http.Post(fmt.Sprintf("http://%s/join", joinAddr), "application-type/json", bytes.NewReader(b))
	if err != nil {
		log.Error(err.Error())
		return ErrNodeJoinMsg
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Error(err.Error())
		}
	}(resp.Body)
	return nil
}

type RMikuKv struct {
	raftInstance *raftkv.RaftKV
}

func (r *RMikuKv) Query(key string) (string, error) {
	return r.raftInstance.Query(key)
}

func (r *RMikuKv) Insert(kvPairs []*persistent.KVPair) error {
	return r.raftInstance.Insert(kvPairs)
}

func (r *RMikuKv) Delete(kvPairs []*persistent.KVPair) error {
	return r.raftInstance.Delete(kvPairs)
}

func (r *RMikuKv) PrefixIterate(prefix []byte, consumeCallback func(k, val []byte) bool) {
	r.raftInstance.PrefixIterate(prefix, consumeCallback)
}

func (r *RMikuKv) IsLeader() bool {
	return r.raftInstance.IsLeader()
}

func (r *RMikuKv) Shutdown() error {
	return r.raftInstance.Shutdown()
}

func (r *RMikuKv) RaftShutdown() error {
	return r.raftInstance.RaftShutdown()
}
