package tcpgossip

import (
	"context"
	"fmt"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/NBSChain/go-nbs/utils"
	"github.com/gogo/protobuf/proto"
	"math/rand"
	"net"
	"time"
)

var (
	conf          *GspConf = nil
	NoTimeOut              = time.Time{}
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
	Condition                int
	UpdateWeightNo           int
	CtrlMsgTimeOut           time.Duration
	RetrySubInterval         time.Duration
	HeartBeat                time.Duration
	ExpireTime               time.Duration
}

type GspCtrlNode struct {
	subNo  int
	ctx    context.Context
	cancel context.CancelFunc

	nodeId      string
	serviceConn *net.TCPListener

	msgTask    chan *gsp_tcp.CtrlMsg
	msgCounter map[string]int

	outView *viewMap
	inView  *viewMap
}

func newGspNode() *GspCtrlNode {

	ctx, cl := context.WithCancel(context.Background())
	node := &GspCtrlNode{
		ctx:        ctx,
		cancel:     cl,
		outView:    newViewMap(),
		inView:     newViewMap(),
		msgCounter: make(map[string]int),
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

func (node *GspCtrlNode) statusMonitor() {

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
			node.msgCounter = make(map[string]int)
			if node.inView.IsEmpty() && !isGenesis {
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
			case gsp_tcp.MsgType_UpdateIV:
				err = node.outView.UpdateLocalWeight(msg)
			case gsp_tcp.MsgType_UpdateOV:
				err = node.inView.UpdateLocalWeight(msg)
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

	go node.statusMonitor()

	go node.msgProcessor()

	select {
	case <-node.ctx.Done():
		logger.Warning("process finish......")
	}

	node.Destroy()
}

func (node *GspCtrlNode) unsubscribe() {

	outs := node.outView.Keys()
	ins := node.inView.Keys()

	lenIn := len(ins) - conf.Condition - 1 - 1
	lenOut := len(outs)
	for i := 0; i < lenIn && lenOut > 0; i++ {

		inItem, _ := node.inView.Value(ins[i])

		outItem, _ := node.outView.Value(outs[i%lenOut])

		inItem.send(node.ReplaceMsg(outItem.nodeID, outItem.peerIP))
	}

	for i := lenIn; i >= 0 && i < len(ins); i++ {
		inItem, _ := node.inView.Value(ins[i])
		inItem.send(node.RemoveMsg())
	}

}

func (node *GspCtrlNode) Destroy() {

	node.unsubscribe()

	node.outView.Clear()
	node.inView.Clear()

	node.msgCounter = nil
	close(node.msgTask)
}

func (node *GspCtrlNode) connReceiver() {

	logger.Info("tcp gossip node prepare to join in the new network.....")

	defer node.serviceConn.Close()
	defer node.cancel()

	for {
		conn, err := node.serviceConn.Accept()
		if err != nil {
			logger.Warning("service is done:->", err)
			return
		}
		logger.Debug("one connHandle create:->", conn.RemoteAddr().String())
		node.connHandle(conn)
	}
}

func (node *GspCtrlNode) connHandle(conn net.Conn) {

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
}

func (node *GspCtrlNode) getForward(msg *gsp_tcp.CtrlMsg) error {
	forward := msg.Forward
	nodeId := forward.NodeId

	if node.msgCounter[forward.MsgId]++; node.msgCounter[forward.MsgId] >= 10 {
		logger.Warning("forwarded too many times, and discard :->", forward)
		return nil
	}

	prob := float64(1) / float64(1+node.outView.Size())
	randProb := rand.Float64()

	_, ok := node.outView.Value(nodeId)
	if ok || randProb > prob {
		item := node.outView.ChoseRandom()
		logger.Debug("introduce you to my friend:->", item.nodeID, ok, randProb, prob)
		data, _ := proto.Marshal(msg)
		return item.send(data)
	}

	if nodeId == node.nodeId {
		return ESelfReq
	}

	return node.acceptForwarded(msg)
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
	node.outView.Add(nodeId, e)
	node.ShowViews()

	logger.Debug("welcome you:->", forward)

	return nil
}
