package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/NBSChain/go-gossip-tcp"
	"github.com/NBSChain/go-nbs/utils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var rootCmd = &cobra.Command{
	Use: "gsp",

	Short: "gsp is a basic gossip protocol implementation written by tcp.",

	Long: "gsp is a basic gossip protocol implementation written by tcp.",

	Run: mainRun,
}

var genesisIp string
var servicePort int

func init() {
	rootCmd.Flags().StringVarP(&genesisIp, "genesis", "g", "52.8.190.235", "set the genesis ip address")
	rootCmd.Flags().IntVarP(&servicePort, "port", "p", 13001, "set the genesis service port")
}
func main() {

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func mainRun(cmd *cobra.Command, args []string) {

	msgMaker := NewMsgManager()

	c := &tcpgossip.GspConf{
		NodeId:                   msgMaker.nodeId,
		GossipControlMessageSize: 1 << 12,
		GenesisIP:                genesisIp,
		TCPServicePort:           servicePort,
		MaxViewItem:              20,
		Condition:                1,
		UpdateWeightNo:           10,
		CtrlMsgTimeOut:           time.Second * 4,
		RetrySubInterval:         time.Second * 15,
		HeartBeat:                time.Second * 100,
		ExpireTime:               time.Hour, //time.Second * 300
	}

	node := tcpgossip.GetInstance()
	if err := node.Init(c); err != nil {
		panic(err)
	}
	node.Run()
}

type msgManager struct {
	nodeId string
	uKey   string
}

const (
	configFile    = "gsp.dat"
	dataSeparator = "++++++"
)

func NewMsgManager() msgManager {

	manager := &msgManager{}

	if _, ok := utils.FileExists(configFile); !ok {
		manager.createNodeId()
	} else {
		manager.readNodeId()
	}

	return *manager
}

func (msg *msgManager) createNodeId() {

	file, err := os.Create(configFile)
	if err != nil {
		panic(err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 256)
	if err != nil {
		panic(err)
	}

	publicKey := &privateKey.PublicKey
	pubData := x509.MarshalPKCS1PublicKey(publicKey)
	nodeId := base64.StdEncoding.EncodeToString(pubData)

	priData := x509.MarshalPKCS1PrivateKey(privateKey)
	uKey := base64.StdEncoding.EncodeToString(priData)

	data := strings.Join([]string{nodeId, uKey}, dataSeparator)

	if _, err := file.Write([]byte(data)); err != nil {
		panic(err)
	}

	msg.nodeId = nodeId
	msg.uKey = uKey
}

func (msg *msgManager) readNodeId() {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	data := strings.Split(string(b), dataSeparator)
	msg.nodeId = data[0]
	msg.uKey = data[1]
}
