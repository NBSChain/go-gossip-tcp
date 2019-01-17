package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"github.com/NBSChain/go-nbs/utils"
	"io/ioutil"
	"os"
	"strings"
)

type msgManager struct {
	nodeId string
	uKey   string
}

const (
	configFile    = ".bat"
	dataSeparator = "++++++"
)

func NewMsgManager() msgManager {

	manager := &msgManager{}

	if _, ok := utils.FileExists(".bat"); !ok {
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

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
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
