package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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

type ConnectData struct {
	Candidates []ICECandidate
	Uflag string
	Pass string
}

var useSCTP = true

func main() {
	reader := bufio.NewReader(os.Stdin)
	uploading := len(os.Args) >= 2

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

	err = agent.OnConnectionStateChange(func(state ice.ConnectionState) {
		fmt.Printf("State Change: %s\n", state.String())

		if state == ice.ConnectionStateConnected {
			fmt.Printf("Connected!\n")
		}
	})

	myCandidates, err := agent.GetLocalCandidates()
	myIceCandidates, err := newICECandidatesFromICE(myCandidates)
	checkError(err)
	uflag, pass := agent.GetLocalUserCredentials()
	myConnectData := &ConnectData{
		Candidates: myIceCandidates,
		Uflag: uflag,
		Pass: pass,
	}
	myConnectDataJson, err := json.Marshal(myConnectData)
	checkError(err)

	fmt.Printf("Candidates:\n%s\n", myConnectDataJson)
	//connectDataJson = base64.StdEncoding.EncodeToString(connectDataJson)


	connectDataJson, err := reader.ReadString('\n')
	checkError(err)

	connectData := &ConnectData{}
	err = json.Unmarshal([]byte(connectDataJson), connectData)
	checkError(err)

	for _, c := range connectData.Candidates {
		i, err := c.toICE()
		checkError(err)

		err = agent.AddRemoteCandidate(i)
		checkError(err)
	}

	var conn *ice.Conn
	if uploading {
		conn, err = agent.Dial(context.Background(), connectData.Uflag, connectData.Pass)
	} else {
		conn, err = agent.Accept(context.Background(), connectData.Uflag, connectData.Pass)
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

		var stream *sctp.Stream
		if uploading {
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

	var f *os.File
	if uploading {
		var err error
		f, err = os.Open(os.Args[1])
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
	defer  f.Close()
	
	if uploading {
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