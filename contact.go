package tcpgossip

import (
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"math/rand"
	"net"
	"time"
)

func (node *GspCtrlNode) asGenesisNode(msg *gsp_tcp.CtrlMsg, conn net.Conn) error {
	defer conn.Close()
	logger.Debug("oh, I am the Genesis node:->", msg)

	if _, err := conn.Write(node.SubAckMSg()); err != nil {
		logger.Warning("write sub ack back err :->", err)
		return err
	}

	nodeId := msg.Subscribe.NodeId
	if nodeId == node.nodeId {
		return ESelfReq
	}

	ttl := int32(len(node.outView))
	rAddr := conn.RemoteAddr().String()
	ip, _, _ := net.SplitHostPort(rAddr)

	if ttl == 0 {
		return node.asContactNode(nodeId, ip)
	}

	return node.voteTheContact(node.VoteMSG(nodeId, ip, ttl))
}

func (node *GspCtrlNode) voteTheContact(data []byte) error {

	node.normalizeProbability()

	item := node.getRandomNodeByProb()

	logger.Debug("vote the right contact:->", item.peerID)

	return item.send(data)
}

func (node *GspCtrlNode) asContactNode(nodeId, ip string) error {

	if _, ok := node.outView[nodeId]; ok {
		return EDuplicateSub
	}

	node.broadCast(nodeId, ip)

	if err := node.notifyApplier(nodeId, ip); err != nil {
		return err
	}

	return nil
}

func (node *GspCtrlNode) getVote(msg *gsp_tcp.CtrlMsg) error {
	vote := msg.Vote
	ttl := vote.TTL - 1
	if ttl <= 0 {
		return node.asContactNode(vote.NodeId, vote.IP)
	}

	data := node.VoteMSG(vote.NodeId, vote.IP, ttl)
	return node.voteTheContact(data)
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
		conn.Close()
		return err
	}

	e := newViewEntity(conn, ip, nodeId, node.msgTask)
	node.outLock.Lock()
	node.outView[nodeId] = e
	node.outLock.Unlock()

	node.inLock.Lock()
	node.inView[nodeId] = e
	node.inLock.Unlock()

	e.probability = node.averageProbability()

	node.ShowViews()

	return nil
}

func (node *GspCtrlNode) averageProbability() float64 {
	if len(node.outView) == 0 {
		return 1.0
	}

	var sum float64
	for _, item := range node.outView {
		sum += item.probability
	}

	return sum / float64(len(node.outView))
}

func (node *GspCtrlNode) normalizeProbability() {

	if len(node.outView) == 0 {
		return
	}

	var sum float64
	for _, item := range node.outView {
		sum += item.probability
	}

	for _, item := range node.outView {
		item.probability = item.probability / sum
	}
}

func (node *GspCtrlNode) getRandomNodeByProb() *ViewEntity {

	rand.Seed(time.Now().UnixNano())

	var (
		p           = rand.Float64()
		sum         = 0.0
		index       = 0
		defaultNode *ViewEntity
	)

	for _, item := range node.outView {

		sum += item.probability
		if p < sum {
			return item
		}
		if index == 0 {
			defaultNode = item
		}

		index++
	}

	return defaultNode
}

func (node *GspCtrlNode) choseRandom() *ViewEntity {

	idx := rand.Intn(len(node.outView))
	i := 0

	for _, item := range node.outView {
		if i == idx {
			return item
		}
		i++
	}
	return nil
}

func (node *GspCtrlNode) acceptForwarded(msg *gsp_tcp.CtrlMsg) error {
	return nil
}
