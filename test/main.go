package main

import (
	"fmt"
	"github.com/NBSChain/go-gossip-tcp"
	"github.com/spf13/cobra"
	"os"
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
	node := tcpgossip.NewGspNode(msgMaker.nodeId)
	c := &tcpgossip.GspConf{
		GossipControlMessageSize: 1 << 12,
		GenesisIP:                genesisIp,
		TCPServicePort:           servicePort,
		SubTimeOut:               time.Second * 4,
	}

	if err := node.Init(c); err != nil {
		panic(err)
	}

	node.Run()
}
