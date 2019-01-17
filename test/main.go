package main

import (
	"fmt"
	"github.com/NBSChain/go-gossip-tcp"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use: "gsp",

	Short: "gsp is a basic gossip protocol implementation written by tcp.",

	Long: "gsp is a basic gossip protocol implementation written by tcp.",

	Run: mainRun,
}

func main() {

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func mainRun(cmd *cobra.Command, args []string) {

	node := tcpgossip.NewGspNode()

	if err := node.Init(); err != nil {
		panic(err)
	}

	node.Run()
}
