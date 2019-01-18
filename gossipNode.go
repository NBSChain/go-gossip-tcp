package tcpgossip

import (
	"context"
	"fmt"
	"github.com/NBSChain/go-gossip-tcp/pbs"
	"github.com/NBSChain/go-nbs/utils"
	"github.com/gogo/protobuf/proto"
	"net"
	"time"
)

var (
	conf     *GspConf = nil
	logger            = utils.GetLogInstance()
	ESelfReq          = fmt.Errorf("it's a soliloquy")
)

type GspConf struct {
	NodeId                   string
	GenesisIP                string //
	TCPServicePort           int    //= 13001
	GossipControlMessageSize int
	MaxViewItem              int
	SubTimeOut               time.Duration
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
	outView     map[string]*ViewEntity
	inView      map[string]*ViewEntity
}

func newGspNode() *GspCtrlNode {
	ctx, cl := context.WithCancel(context.Background())

	node := &GspCtrlNode{
		ctx:     ctx,
		cancel:  cl,
		msgTask: make(chan *gsp_tcp.CtrlMsg, conf.MaxViewItem),
		outView: make(map[string]*ViewEntity),
		inView:  make(map[string]*ViewEntity),
	}
	return node
}

func (node *GspCtrlNode) Init(c *GspConf) error {
	conf = c
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

func (node *GspCtrlNode) Run() {

	go node.connReceiver()

	go node.gossipManager()

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
		err = node.asGenesisNode(msg, conn)
	case gsp_tcp.MsgType_GotContact:
		err = node.subSuccess(msg, conn)
	}

	logger.Debug("one connHandle exit:->", conn.RemoteAddr().String(), err)
}
