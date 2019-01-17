package tcpgossip

import (
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"net"
)

func (node *GspCtrlNode) asGenesisNode(msg *gsp_tcp.CtrlMsg, conn net.Conn) error {

	logger.Debug("oh, I am the Genesis node:->", msg)

	if _, err := conn.Write(node.AckMSG()); err != nil {
		logger.Warning("write sub ack back err :->", err)
		return err
	}

	nodeId := msg.Subscribe.NodeId
	ttl := int32(len(node.outView))
	rAddr := conn.RemoteAddr().String()
	ip, _, _ := net.SplitHostPort(rAddr)

	if ttl == 0 {
		return node.asContactNode(nodeId, ip)
	}

	return node.voteTheContact(node.VoteMSG(nodeId, ip, ttl))
}

func (node *GspCtrlNode) voteTheContact(data []byte) error {
	return nil
}

func (node *GspCtrlNode) asContactNode(nodeId, ip string) error {

	node.broadCast(nodeId, ip)

	if err := node.notifyApplier(nodeId, ip); err != nil {
		return err
	}

	return nil
}

func (node *GspCtrlNode) broadCast(nodeId, ip string) {
	logger.Debug("prepare to introduce you:->", nodeId, ip)
	if len(node.outView) == 0 {
		logger.Debug("I have no friends to introduce to you")
		return
	}

	data := node.FwdSubMSG(nodeId, ip)
	for id, e := range node.outView {
		if _, err := e.conn.Write(data); err != nil {
			logger.Warning("introduce new member err:->", err, id)
			node.removeViewEntity(id)
		}
		logger.Debug("introduce new subscriber to my friend:->", id)
	}
}

func (node *GspCtrlNode) notifyApplier(nodeId, ip string) error {

	logger.Debug("accept you and save infos:->", nodeId, ip)

	conn, err := net.DialTCP("tcp4", nil, &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: conf.TCPServicePort,
	})

	if err != nil {
		logger.Warning("failed to act as contact node:->", err)
		return err
	}

	myIp, _, _ := net.SplitHostPort(conn.LocalAddr().String())

	if _, err := conn.Write(node.ContactMsg(myIp)); err != nil {
		logger.Warning("err when notify the subscriber:->", err)
		return err
	}

	e := NewViewEntity(conn)
	node.outView[nodeId] = e
	node.inView[nodeId] = e

	return nil
}
