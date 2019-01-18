package tcpgossip

import (
	"context"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/gogo/protobuf/proto"
	"net"
)

type ViewEntity struct {
	ok     bool
	peerID string
	peerIP string
	ctx    context.Context
	closer context.CancelFunc
	conn   net.Conn
}

func (e *ViewEntity) reading() {

	logger.Debug("start to read......")
	defer e.Close()

	for {

		buffer := make([]byte, conf.GossipControlMessageSize)
		n, err := e.conn.Read(buffer)

		if err != nil {
			logger.Warning("connection node read err:->", e.peerID)
			return
		}

		msg := &gsp_tcp.CtrlMsg{}
		if err := proto.Unmarshal(buffer[:n], msg); err != nil {
			return
		}

		logger.Debug("connection node received control msgManager:->", msg)
	}
}

func NewViewEntity(c net.Conn, ip, id string) *ViewEntity {

	ctx, cancel := context.WithCancel(context.Background())
	node := &ViewEntity{
		ctx:    ctx,
		closer: cancel,
		conn:   c,
		ok:     true,
		peerID: id,
		peerIP: ip,
	}

	go node.reading()
	return node
}

func (e *ViewEntity) Close() {

	if !e.ok {
		logger.Debug("try to cancel a closed connection node:->", e.peerID)
		return
	}

	logger.Info("the connection node closed:->", e.peerID)

	e.ok = false
	e.closer()

	if err := e.conn.Close(); err != nil {
		logger.Warning("failed to cancel connection node:->", e.peerID)
	}
}

func (node *GspCtrlNode) removeViewEntity(id string) {

	if item, ok := node.outView[id]; ok {
		logger.Debug("remove from out put view :->", item.peerID)
		delete(node.outView, id)
		item.Close()
	}

	if item, ok := node.inView[id]; ok {
		logger.Debug("remove from in put view :->", item.peerID)
		delete(node.outView, id)
		item.Close()
	}
}

func (node *GspCtrlNode) sendHeartBeat() {

	data := node.HeartBeatMsg()

	for id, item := range node.outView {

		if _, err := item.conn.Write(data); err != nil {
			logger.Warning("sending heart beat err:->", id, err)
			node.removeViewEntity(id)
		}
	}
}
