package tcpgossip

import (
	"context"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/gogo/protobuf/proto"
	"net"
)

type ViewEntity struct {
	ok     bool
	peerId string
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
			logger.Warning("connection node read err:->", e.peerId)
			return
		}

		msg := &gsp_tcp.CtrlMsg{}
		if err := proto.Unmarshal(buffer[:n], msg); err != nil {
			return
		}

		logger.Debug("connection node received control msgManager:->", msg)
	}
}

func NewViewEntity(c net.Conn) *ViewEntity {

	ctx, cancel := context.WithCancel(context.Background())
	node := &ViewEntity{
		ctx:    ctx,
		closer: cancel,
		conn:   c,
		ok:     true,
	}

	go node.reading()
	return node
}

func (e *ViewEntity) Close() {

	if !e.ok {
		logger.Debug("try to close a closed connection node:->", e.peerId)
		return
	}

	logger.Info("the connection node closed:->", e.peerId)

	e.ok = false
	e.closer()

	if err := e.conn.Close(); err != nil {
		logger.Warning("failed to close connection node:->", e.peerId)
	}
}

func (node *GspCtrlNode) removeViewEntity(id string) {

	if item, ok := node.outView[id]; ok {
		delete(node.outView, id)
		item.Close()
	}

	if item, ok := node.inView[id]; ok {
		delete(node.outView, id)
		item.Close()
	}
}
