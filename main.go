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

var stun = "stun:127.0.0.1:3478"
//var stun = "stun:stun.l.google.com:19302"

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

func exchangeManually(data ExchangeData) *ExchangeData {
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
//var exchange = exchangeManually

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

	stunUrl, err := ice.ParseURL(stun)
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

	var connIO io.ReadWriteCloser
	err = agent.OnConnectionStateChange(func(state ice.ConnectionState) {
		fmt.Printf("State Change: %s\n", state.String())
		if state == ice.ConnectionStateDisconnected {
			if connIO != nil {
				err := connIO.Close()
				checkError(err)
			}
		}
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

	var conn *ice.Conn
	if uploadingFile != "" {
		conn, err = agent.Accept(context.Background(), partnerData.Uflag, partnerData.Pass)
	} else {
		conn, err = agent.Dial(context.Background(), partnerData.Uflag, partnerData.Pass)
	}
	checkError(err)
	defer conn.Close()

	go func() {
		if uploadingFile != "" {
			conn.Write([]byte("hello"))
		} else {
			conn.Write([]byte("world"))
		}
	}()

	buffer := make([]byte, 32 * 1000)
	conn.Read(buffer)

	fmt.Printf("Receive msg: %s\n", string(buffer))

	if useSCTP {
		var association *sctp.Association
		config := sctp.Config{
			NetConn: conn,
			LoggerFactory: logging.NewDefaultLoggerFactory(),
			MaxReceiveBufferSize: 10 * 1024 * 1024,
		}
		if uploadingFile != "" {
			association, err = sctp.Client(config)
		} else {
			association, err = sctp.Server(config)
		}
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
		time.Sleep(2 * time.Second)
		fmt.Printf("Uploading....\n")
	} else {
		fmt.Printf("Downloading....\n")
	}

	if uploadingFile != "" {
		var n int64
		if useSCTP {
			n, err = io.Copy(connIO, f)
			//n = copy(connIO, f)
		} else {
			//n = copy(connIO, f)
			n, err = io.CopyBuffer(connIO, f, make([]byte, 5 * 1200))
		}
		checkError(err)

		fmt.Printf("Success %v bytes sent!\n", n)
		connIO.Close()
	} else {
		n, err := io.Copy(f, connIO)
		checkError(err)

		//n := copy(f, connIO)

		fmt.Printf("Saved %v bytes!\n", n)
		connIO.Close()
	}
}

func copy(dst io.Writer, src io.Reader) int64 {
	buffer := make([]byte, 32 * 1000)
	var n int64
	for {
		_, err := src.Read(buffer)
		if err == io.EOF {
			break
		}
		checkError(err)
		i, err := dst.Write(buffer)
		n = n + int64(i)
		if err == io.EOF {
			break
		}
		checkError(err)
	}

	return n
}