package tcpgossip

import (
	"context"
	"fmt"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/gogo/protobuf/proto"
	"net"
	"sync"
	"time"
)

type ViewEntity struct {
	sync.RWMutex
	ok bool

	nodeID string
	peerIP string

	ctx      context.Context
	closer   context.CancelFunc
	conn     net.Conn
	pareNode *GspCtrlNode

	probability   float64
	heartBeatTime time.Time
	expiredTime   time.Time
}

func (e *ViewEntity) reading() {

	logger.Debug("start to read......")
	defer e.pareNode.removeViewEntity(e.nodeID)

	for {

		buffer := make([]byte, conf.GossipControlMessageSize)
		n, err := e.conn.Read(buffer)

		if err != nil {
			logger.Warning("connection node read err:->", e.nodeID, err)
			return
		}

		msg := &gsp_tcp.CtrlMsg{}
		if err := proto.Unmarshal(buffer[:n], msg); err != nil {
			return
		}

		logger.Debug("view entity node received :->", msg)

		if msg.Type == gsp_tcp.MsgType_HeartBeat {
			e.Lock()
			e.heartBeatTime = time.Now()
			e.Unlock()

		} else {
			e.pareNode.msgTask <- msg
		}
	}
}

func (node *GspCtrlNode) newViewEntity(c net.Conn, ip, id string) *ViewEntity {

	logger.Debug("create a new item :->", ip, id)
	ctx, cancel := context.WithCancel(context.Background())
	e := &ViewEntity{
		pareNode:      node,
		probability:   node.averageProbability(),
		ctx:           ctx,
		closer:        cancel,
		conn:          c,
		ok:            true,
		nodeID:        id,
		peerIP:        ip,
		expiredTime:   time.Now().Add(conf.ExpireTime),
		heartBeatTime: time.Now(),
	}

	go e.reading()
	return e
}

func (e *ViewEntity) send(msg []byte) error {

	if _, err := e.conn.Write(msg); err != nil {
		logger.Warning("send msg err :->", err)
		e.pareNode.removeViewEntity(e.nodeID)
		return err
	}
	return nil
}

func (e *ViewEntity) Close() {
	e.Lock()
	defer e.Unlock()

	if !e.ok {
		logger.Debug("try to cancel a closed connection node:->", e.nodeID)
		return
	}
	logger.Info("the connection node closed:->", e.nodeID)

	e.ok = false
	e.closer()
	if err := e.conn.Close(); err != nil {
		logger.Warning("failed to cancel connection node:->", e.nodeID)
	}
}

func (node *GspCtrlNode) removeViewEntity(id string) {
	if item, ok := node.outView[id]; ok {
		logger.Debug("remove from out put view :->", item.nodeID)
		delete(node.outView, id)
		item.Close()
	}

	if item, ok := node.inView[id]; ok {
		logger.Debug("remove from in put view :->", item.nodeID)
		delete(node.inView, id)
		item.Close()
	}

	node.ShowViews()

	if len(node.inView) == 0 {
		if err := node.Subscribe(node.SubMsg(true)); err != nil {
			logger.Warning("resubscribe err:->", err)
		}
		logger.Debug("no input view entities and resubscribe now")
	}
}

func (node *GspCtrlNode) sendHeartBeat() {

	data := node.HeartBeatMsg()
	now := time.Now()

	for id, item := range node.outView {

		if now.After(item.expiredTime) {
			logger.Warning("subscribe expired:->", id)
			node.removeViewEntity(id)
			continue
		}
		item.send(data)
	}
}

func (node *GspCtrlNode) updateWeight() {

	var sum float64
	for _, item := range node.inView {
		sum += item.probability
	}

	for _, item := range node.inView {
		item.probability = item.probability / sum
		item.send(node.UpdateMsg(gsp_tcp.MsgType_UpdateIV, item.probability))
	}

	sum = 0.0
	for _, item := range node.outView {
		sum += item.probability
	}

	for _, item := range node.outView {
		item.probability = item.probability / sum
		item.send(node.UpdateMsg(gsp_tcp.MsgType_UpdateOV, item.probability))
	}
	logger.Debug("time to update arc weight.......")
}

func (node *GspCtrlNode) updateOutViewWeight(msg *gsp_tcp.CtrlMsg) error {
	up := msg.UpdateWeight
	nodeId := up.NodeId

	item, ok := node.outView[nodeId]
	if !ok {
		return fmt.Errorf("update out view err, no such item(%s):->", item.KeyString())
	}

	item.probability = up.Weight
	logger.Debug("item in out view get updated:->", item.KeyString())

	return nil
}

func (node *GspCtrlNode) updateInViewWeight(msg *gsp_tcp.CtrlMsg) error {
	up := msg.UpdateWeight
	nodeId := up.NodeId

	item, ok := node.inView[nodeId]
	if !ok {
		return fmt.Errorf("update in view err, no such item(%s):->", item.KeyString())
	}

	item.probability = up.Weight
	logger.Debug("item in in view get updated:->", item.KeyString())

	return nil
}
