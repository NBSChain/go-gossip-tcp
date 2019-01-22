package tcpgossip

import (
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"net"
)

func (node *GspCtrlNode) asProxyNode(msg *gsp_tcp.CtrlMsg, conn net.Conn) error {
	defer conn.Close()
	logger.Debug("oh, I am the proxy node:->", msg)

	if _, err := conn.Write(node.SubAckMSg()); err != nil {
		logger.Warning("write sub ack back err :->", err)
		return err
	}

	nodeId := msg.Subscribe.NodeId
	if nodeId == node.nodeId {
		return ESelfReq
	}

	if node.subNo++; node.subNo >= conf.UpdateWeightNo {
		go node.updateWeight()
		node.subNo = 0
	}

	ttl := int32(len(node.outView)) * 2
	rAddr := conn.RemoteAddr().String()
	ip, _, _ := net.SplitHostPort(rAddr)

	if ttl == 0 {
		logger.Debug("no more friends and I am the only can be a contact:->", nodeId)
		return node.asContactNode(nodeId, ip)
	}

	return node.voteTheContact(node.VoteMSG(nodeId, ip, ttl))
}

func (node *GspCtrlNode) voteTheContact(data []byte) error {

	node.normalizeProbability()

	item := node.getRandomNodeByProb()

	logger.Debug("vote a contact:->", item.nodeID)

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
		logger.Debug("I am your destiny:->", vote)
		return node.asContactNode(vote.NodeId, vote.IP)
	}

	data := node.VoteMSG(vote.NodeId, vote.IP, ttl)
	return node.voteTheContact(data)
}

func (node *GspCtrlNode) broadCast(nodeId, ip string) {

	if len(node.outView) == 0 {
		logger.Debug("as contact I have no friends to introduce to you")
		return
	}

	data := node.FwdSubMSG(nodeId, ip)

	for id, e := range node.outView {
		e.send(data)
		logger.Debug("introduce new subscriber to my friend:->", id)
	}

	for i := 0; i < conf.Condition; i++ {
		item := node.choseRandom()
		item.send(data)
	}
}

func (node *GspCtrlNode) notifyApplier(nodeId, ip string) error {

	logger.Debug("accept you as contact and save infos:->", nodeId, ip)

	conn, err := node.pingMsg(nil, &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: conf.TCPServicePort,
	}, 0, node.ContactMsg())
	if err != nil {
		logger.Warning("failed to act as contact node:->", err)
		return err
	}

	e := node.newViewEntity(conn, ip, nodeId)

	node.outView[nodeId] = e
	node.inView[nodeId] = e

	node.ShowViews()

	return nil
}
