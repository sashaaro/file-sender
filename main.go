package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/ice"
	"github.com/pion/logging"
	"github.com/pion/sctp"
	"io"
	"os"
	"strings"
	"time"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

type ExchangeData struct {
	Candidates []ICECandidate
	Uflag string
	Pass string
}

var useSCTP = true

var webSocketUrl = "ws://localhost:8443/ws"

func exchangeSocket(data ExchangeData) *ExchangeData {
	c, _, err := websocket.DefaultDialer.Dial(webSocketUrl, nil)
	checkError(err)

	err = c.WriteJSON(data)
	checkError(err)

	partnerData := &ExchangeData{}
	err = c.ReadJSON(partnerData)
	checkError(err)

	return partnerData
}

func exchangeMannualy(data ExchangeData) *ExchangeData {
	myConnectDataJson, err := json.Marshal(data)
	checkError(err)
	fmt.Printf("Candidates:\n%s\n", myConnectDataJson)
	//connectDataJson = base64.StdEncoding.EncodeToString(connectDataJson)
	connectDataJson, err := bufio.NewReader(os.Stdin).ReadString('\n')
	checkError(err)

	partnerData := &ExchangeData{}
	err = json.Unmarshal([]byte(connectDataJson), partnerData)
	checkError(err)

	return partnerData
}

var exchange = exchangeSocket

func main() {
	reader := bufio.NewReader(os.Stdin)
	args := os.Args[1:]

	var uploadingFile string
	if len(args) == 1 {
		if args[0] == "server" {
			signalingServer()
			return
		} else {
			uploadingFile = args[0]
		}
	}

	stunUrl, err := ice.ParseURL("stun:stun.l.google.com:19302")
	candidateSelectionTimeout := 30 * time.Second
	connectionTimeout := 5 * time.Second
	config := &ice.AgentConfig{
		Urls: []*ice.URL{stunUrl},
		NetworkTypes: []ice.NetworkType{
			ice.NetworkTypeUDP4,
			ice.NetworkTypeTCP4,
		},
		CandidateTypes: []ice.CandidateType{
			ice.CandidateTypeHost,
			ice.CandidateTypeServerReflexive,
			ice.CandidateTypePeerReflexive,
			ice.CandidateTypeRelay,
		},
		CandidateSelectionTimeout: &candidateSelectionTimeout,
		ConnectionTimeout: &connectionTimeout,
	}

	agent, err := ice.NewAgent(config)
	checkError(err)

	defer agent.Close()

	err = agent.OnConnectionStateChange(func(state ice.ConnectionState) {
		fmt.Printf("State Change: %s\n", state.String())
	})

	myCandidates, err := agent.GetLocalCandidates()
	myIceCandidates, err := newICECandidatesFromICE(myCandidates)
	checkError(err)
	uflag, pass := agent.GetLocalUserCredentials()

	partnerData := exchange(ExchangeData{
		Candidates: myIceCandidates,
		Uflag: uflag,
		Pass: pass,
	})

	for _, c := range partnerData.Candidates {
		i, err := c.toICE()
		checkError(err)

		err = agent.AddRemoteCandidate(i)
		checkError(err)
	}


	var f *os.File
	if uploadingFile != "" {
		fmt.Printf("Uploading %s\n", uploadingFile)

		var err error
		f, err = os.Open(uploadingFile)
		checkError(err)
	} else {
		for {
			fmt.Printf("Save to: ")

			filePath, err := reader.ReadString('\n')
			checkError(err)
			f, err = os.Create(strings.Trim(filePath, "\n\r"))

			if err == nil {
				break
			} else {
				fmt.Printf(err.Error())
			}
		}
	}
	defer f.Close()


	var conn *ice.Conn
	if uploadingFile != "" {
		conn, err = agent.Dial(context.Background(), partnerData.Uflag, partnerData.Pass)
	} else {
		conn, err = agent.Accept(context.Background(), partnerData.Uflag, partnerData.Pass)
	}
	checkError(err)
	defer conn.Close()


	var connIO io.ReadWriter
	if useSCTP {
		association, err := sctp.Client(sctp.Config{
			NetConn: conn,
			LoggerFactory: logging.NewDefaultLoggerFactory(),
		})
		checkError(err)

		defer association.Close()

		var stream *sctp.Stream
		if uploadingFile != "" {
			stream, err = association.OpenStream(777, sctp.PayloadTypeWebRTCBinary)
		} else {
			stream, err = association.AcceptStream()
		}
		checkError(err)

		defer stream.Close()

		connIO = stream
	} else {
		connIO = conn
	}

	if uploadingFile != "" {
		time.Sleep(5 * time.Second)
		fmt.Printf("Uploading....\n")
	} else {
		fmt.Printf("Downloading....\n")
	}

	if uploadingFile != "" {
		var n int64
		if useSCTP {
			n, err = io.Copy(connIO, f)
		} else {
			//n, err := io.CopyBuffer(conn, f, make([]byte, 1200))
			n, err = io.CopyBuffer(connIO, f, make([]byte, 5 * 1200))
		}
		checkError(err)

		fmt.Printf("Success %v bytes sent!\n", n)
	} else {
		_, err = io.Copy(f, connIO)
		checkError(err)

		fmt.Printf("Saved!\n")
	}
}