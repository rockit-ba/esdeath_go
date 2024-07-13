package main

import (
	esdeathproto "esdeath_go/esdeath_proto"
	"esdeath_go/miku/persistent"
	"fmt"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"strings"
)

const (
	// 用于分割key字段的特殊字符
	keyDelimiter     = "$"
	defConsumerGroup = "default"
)

// ->消息被新增的时候，状态为waitConsume
// ->被一个client 拉取之后，ack之前，状态为consuming
// ->ack之后，消息删除
const (
	waitConsume msgStatus = '0'
	consuming   msgStatus = '1'
)

func produceReqHandle(add *esdeathproto.DelayMsgAdd) (*esdeathproto.AddDelayMsgResult, error) {
	if err := check(add); err != nil {
		return nil, err
	}
	produceResult := &esdeathproto.AddDelayMsgResult{BaseResult: successResult()}

	var pairs []*persistent.KVPair
	// 设置消息状态为待消费，消息状态存在于payload，而不是key中
	statusPayload := waitConsume.toString() + add.Payload
	// 如果一个topic有多个consumerGroup，那么消息会被镜像多份，为了实现同一个topic的不同consumerGroup消费相同的消息
	if cgs, ok := topicToManyConsumerGroupsMap.Get(add.Topic); ok {
		for _, consumeGroup := range cgs.Keys() {
			message := persistent.NewKVPair(
				newKeyEntityFromAddMsg(add, consumeGroup).toStr(),
				statusPayload)
			pairs = append(pairs, message)
		}
	} else {
		message := persistent.NewKVPair(
			newKeyEntityFromAddMsg(add, defConsumerGroup).toStr(),
			statusPayload)
		pairs = append(pairs, message)
	}

	if err := rkv.Insert(pairs); err != nil {
		log.Errorln("insert message error: ", err)
		return nil, err
	}
	// 注意这个返回的msgId 用来被client cancel，而不能被用来ack。
	// cancel的时候会删除当前topic下所有的消息，包括不同consumerGroup的消息。
	produceResult.MsgId = newKeyEntityFromAddMsg(add, "").toStr()
	return produceResult, nil
}

func check(add *esdeathproto.DelayMsgAdd) error {
	// keyDelimiter 符号校验
	if strings.Contains(add.Topic, keyDelimiter) {
		return fmt.Errorf("topic contains special character: %s", keyDelimiter)
	}
	if strings.Contains(add.Tag, keyDelimiter) {
		return fmt.Errorf("tag contains special character: %s", keyDelimiter)
	}
	if strings.Contains(add.MsgId, keyDelimiter) {
		return fmt.Errorf("msgId contains special character: %s", keyDelimiter)
	}
	return nil
}

// 对应kv持久化的key字符串对象
// 字符消息ID: toStr()
type keyEntity struct {
	topic         string
	consumerGroup string
	delayTime     int64
	tag           string
	uniqueId      string
}

// 输出规则：topic + consumerGroup + 延迟时间 + tag + 唯一ID
func (k keyEntity) toStr() string {
	parts := []string{k.topic}
	if k.consumerGroup != "" {
		parts = append(parts, k.consumerGroup)
	}
	parts = append(parts, strconv.FormatInt(k.delayTime, 10), k.tag, k.uniqueId)
	return strings.Join(parts, keyDelimiter)
}

func newKeyEntityFromAddMsg(add *esdeathproto.DelayMsgAdd, consumerGroup string) keyEntity {
	return keyEntity{
		topic:         add.Topic,
		consumerGroup: consumerGroup,
		delayTime:     add.DelayTime,
		tag:           add.Tag,
		uniqueId:      add.MsgId,
	}
}

func newKeyEntityFromBytes(b []byte) (*keyEntity, error) {
	return newKeyEntityFromStr(string(b))
}

const (
	topicIndex         = 0
	consumerGroupIndex = 1
	delayTimeIndex     = 2
	tagIndex           = 3
	uniqueIdIndex      = 4
)

var keyEntityFieldNum = reflect.TypeOf(keyEntity{}).NumField()

func newKeyEntityFromStr(strKey string) (*keyEntity, error) {
	fields := strings.Split(strKey, keyDelimiter)
	if len(fields) < keyEntityFieldNum {
		return nil, fmt.Errorf("invalid key: %s", strKey)
	}
	delayTimeStr := fields[delayTimeIndex]
	delayTime, err := strconv.ParseInt(delayTimeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse delayTime error: %v", err)
	}
	return &keyEntity{
		topic:         fields[topicIndex],
		consumerGroup: fields[consumerGroupIndex],
		delayTime:     delayTime,
		tag:           fields[tagIndex],
		uniqueId:      fields[uniqueIdIndex],
	}, nil
}

type msgStatus byte

func (m msgStatus) toString() string {
	return string(m)
}
func (m msgStatus) toByte() byte {
	return byte(m)
}
