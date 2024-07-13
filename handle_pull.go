package main

import (
	pb "esdeath_go/esdeath_proto"
	"esdeath_go/miku/persistent"
	log "github.com/sirupsen/logrus"
	"time"
)

const defHoldTime = 20 * time.Second

func pullReqHandle(pull *pb.DelayMsgPull) (*PulledMsg, error) {
	var matchEntry *keyEntity
	var matchedKey, matchedValue []byte
	// 非法key将会被移除，因为永远无法消费
	var illegalKeys [][]byte

	// false表示继续遍历，true表示停止遍历
	matchFunc := func(k, v []byte) bool {
		log.Debugln("Iterate key: ", string(k))
		sk, err := newKeyEntityFromBytes(k)
		if err != nil {
			log.Errorln("build key from bytes: ", err)
			illegalKeys = append(illegalKeys, k)
			return false
		}
		// 如果状态不是waitConsume，需要继续遍历
		if msgStatus(v[0]) != waitConsume {
			log.Debugln("msg status is wait consuming")
			return false
		}
		matchEntry = sk
		if sk.delayTime > time.Now().UnixMilli() {
			log.Debugln("delay time not reach: ", sk.delayTime)
			// 遇到第一个延迟时间未到的消息，停止遍历，等待下次长轮询确定后面也不会有达到延迟时间的消息
			return true
		}
		matchedKey, matchedValue = k, v
		return true
	}
	// 避免遍历不是当前topic的key
	prefix := []byte(pull.Topic + keyDelimiter + pull.ConsumerGroup)
	rkv.PrefixIterate(prefix, matchFunc)
	// 删除非法key,当前执行逻辑不影响消费key的逻辑
	go func() {
		if len(illegalKeys) > 0 {
			errPs := make([]*persistent.KVPair, len(illegalKeys))
			for i, errKey := range illegalKeys {
				errPs[i] = persistent.NewKPair(string(errKey))
			}
			if err := rkv.Delete(errPs); err != nil {
				log.Errorln("delete err keys err: ", err)
			}
		}
	}()

	// 没有任何消息：
	//1.当前topic下没有消息 2.当前topic下没有waitConsume状态的消息
	if matchEntry == nil {
		return nil, nil
	}

	// 没有找到达到延迟时间的消息
	if matchedKey == nil {
		return &PulledMsg{DelayTimestamp: matchEntry.delayTime}, nil
	}

	// 更新当前消息状态为 consuming
	matchedValue[0] = consuming.toByte()
	message := persistent.NewKVPair(string(matchedKey), string(matchedValue))
	if err := rkv.Insert([]*persistent.KVPair{message}); err != nil {
		log.Errorln("update consuming status err: ", err)
		return nil, err
	}

	log.Infoln("consume key: ", matchEntry)
	resp := &PulledMsg{
		MsgId:   string(matchedKey),
		Topic:   matchEntry.topic,
		Tag:     matchEntry.tag,
		Payload: matchedValue[1:],
	}
	return resp, nil
}

type PulledMsg struct {
	MsgId          string
	Topic          string
	Tag            string
	Payload        []byte
	DelayTimestamp int64
}
