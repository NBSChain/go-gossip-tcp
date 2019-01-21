package tcpgossip

import (
	"context"
	"fmt"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/NBSChain/go-nbs/utils"
	"github.com/gogo/protobuf/proto"
	"math/rand"
	"net"
	"sync"
	"time"
)

var (
	conf          *GspConf = nil
	logger                 = utils.GetLogInstance()
	ESelfReq               = fmt.Errorf("it's myself")
	EDuplicateSub          = fmt.Errorf("I have accept this sub as contact")
)

type GspConf struct {
	NodeId                   string
	GenesisIP                string //
	TCPServicePort           int    //= 13001
	GossipControlMessageSize int
	MaxViewItem              int
	CtrlMsgTimeOut           time.Duration
	RetrySubInterval         time.Duration
	HeartBeat                time.Duration
	ExpireTime               time.Duration
}

type GspCtrlNode struct {
	ctx    context.Context
	cancel context.CancelFunc

	nodeId      string
	serviceConn *net.TCPListener
	msgTask     chan *gsp_tcp.CtrlMsg
	outLock     sync.RWMutex
	outView     map[string]*ViewEntity
	inLock      sync.RWMutex
	inView      map[string]*ViewEntity
}

func newGspNode() *GspCtrlNode {
	ctx, cl := context.WithCancel(context.Background())

	node := &GspCtrlNode{
		ctx:     ctx,
		cancel:  cl,
		outView: make(map[string]*ViewEntity),
		inView:  make(map[string]*ViewEntity),
	}
	return node
}

func (node *GspCtrlNode) Init(c *GspConf) error {
	conf = c

	node.msgTask = make(chan *gsp_tcp.CtrlMsg, conf.MaxViewItem)
	node.nodeId = c.NodeId

	conn, err := net.ListenTCP("tcp4", &net.TCPAddr{
		Port: conf.TCPServicePort,
	})

	if err != nil {
		logger.Warning("start tcp service failed:->", err)
		return err
	}
	node.serviceConn = conn

	return nil
}

func (node *GspCtrlNode) gossipManager() {

	logger.Info("tcp gossip node prepare to join in the new network.....")

	data := node.SubMsg(false)
	isGenesis := false

ReTry:
	if err := node.Subscribe(data); err != nil {

		logger.Warning("failed to subscribe any contact:->", err)

		if err != ESelfReq {
			time.Sleep(conf.RetrySubInterval)
			goto ReTry
		}

		isGenesis = true
		logger.Info("I'm genesis node......")
	}

	for {
		select {
		case <-time.After(conf.HeartBeat):

			node.sendHeartBeat()

			if len(node.inView) == 0 && !isGenesis {
				data = node.SubMsg(true)
				goto ReTry
			}

		case <-node.ctx.Done():
			logger.Warning("exist the gossip manager thread......")
			return
		}
	}
}

func (node *GspCtrlNode) msgProcessor() {

	for {
		var err error

		select {

		case msg := <-node.msgTask:
			switch msg.Type {
			case gsp_tcp.MsgType_VoteContact:
				err = node.getVote(msg)
			case gsp_tcp.MsgType_Forward:
				err = node.getForward(msg)
			}

		case <-node.ctx.Done():
			logger.Warning("exist message process thread......")
			return
		}

		if err != nil {
			logger.Debug("msg process err:->", err)
		}
	}
}

func (node *GspCtrlNode) Run() {

	go node.connReceiver()

	go node.gossipManager()

	go node.msgProcessor()

	select {
	case <-node.ctx.Done():
		logger.Warning("process finish......")
	}

	node.Destroy()
}

func (node *GspCtrlNode) Destroy() {
}

func (node *GspCtrlNode) connReceiver() {

	logger.Info("tcp gossip node prepare to join in the new network.....")
	defer node.serviceConn.Close()

	for {
		conn, err := node.serviceConn.Accept()
		if err != nil {
			logger.Warning("service is done:->", err)
			node.cancel()
			return
		}

		go node.connHandle(conn)
	}
}

func (node *GspCtrlNode) connHandle(conn net.Conn) {

	logger.Debug("one connHandle create:->", conn.RemoteAddr().String())

	buffer := make([]byte, conf.GossipControlMessageSize)
	n, err := conn.Read(buffer)

	if err != nil {
		logger.Error("get a weak tcp conn:->", err)
		return
	}

	msg := &gsp_tcp.CtrlMsg{}
	if err := proto.Unmarshal(buffer[:n], msg); err != nil {
		logger.Warning("unknown gossip msgManager:->", err)
		return
	}

	switch msg.Type {
	case gsp_tcp.MsgType_SubInit:
		err = node.asProxyNode(msg, conn)
	case gsp_tcp.MsgType_GotContact:
		err = node.subSuccess(msg, conn)
	case gsp_tcp.MsgType_WelCome:
		err = node.beWelcomed(msg, conn)
	}

	logger.Debug("one connHandle exit:->", conn.RemoteAddr().String(), err)
}

func (node *GspCtrlNode) getForward(msg *gsp_tcp.CtrlMsg) error {
	forward := msg.Forward
	nodeId := forward.NodeId

	prob := float64(1) / float64(1+len(node.outView))
	randProb := rand.Float64()

	_, ok := node.outView[nodeId]
	if ok || randProb > prob {
		item := node.choseRandom()
		logger.Debug("introduce you to my friend:->", item.nodeID, ok, randProb, prob)
		data, _ := proto.Marshal(msg)
		return item.send(data)
	}

	if nodeId == node.nodeId {
		return ESelfReq
	}

	return node.acceptForwarded(msg)
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

	logger.Debug("random mode prob:->", p)
	for _, item := range node.outView {

		sum += item.probability
		logger.Debug("total sum, prob:->", sum, item.probability, item.nodeID)

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

	forward := msg.Forward
	nodeId := forward.NodeId

	conn, err := node.pingMsg(nil, &net.TCPAddr{
		IP:   net.ParseIP(forward.IP),
		Port: conf.TCPServicePort,
	}, 0, node.WelcomeMsg())

	if err != nil {
		return err
	}

	e := node.newViewEntity(conn, forward.IP, nodeId)

	node.outLock.Lock()
	node.outView[nodeId] = e
	node.outLock.Unlock()

	node.ShowViews()
	logger.Debug("welcome you->", forward)

	return nil
}
