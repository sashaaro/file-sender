package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pion/ice"
	//"github.com/pion/logging"
	//"github.com/pion/sctp"
	"io"
	"os"
	"context"
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

func main() {
	saveFile := ""

	if len(os.Args) > 2 {
		saveFile = os.Args[1]
	}

	stunUrl, err := ice.ParseURL("stun:stun.l.google.com:19302")
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


	reader := bufio.NewReader(os.Stdin)
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
	if saveFile != "" {
		conn, err = agent.Accept(context.Background(), connectData.Uflag, connectData.Pass)
	} else {
		conn, err = agent.Dial(context.Background(), connectData.Uflag, connectData.Pass)
	}
	checkError(err)

	/*association, err := sctp.Client(sctp.Config{
		NetConn: conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	})
	checkError(err)*/

	if saveFile != "" {
		//stream, err := association.AcceptStream()
		//checkError(err)

		f, err := os.Create(saveFile)
		checkError(err)
		defer  f.Close()

		_, err = io.Copy(f, conn)
		checkError(err)

		fmt.Printf("Saved!\n")
	} else {
		//stream, err := association.OpenStream(777, sctp.PayloadTypeWebRTCBinary)
		//checkError(err)

		fmt.Printf("Input file path: \n")

		filePath, err := reader.ReadString('\n')
		checkError(err)
		f, err := os.Open(filePath)
		checkError(err)
		defer  f.Close()

		buffer := make([]byte, 512)
		var n int
		for {
			_, err = f.Read(buffer)
			if err == io.EOF {
				break
			}
			checkError(err)
			i, err := conn.Write(buffer)
			n = n + i
			if err == io.EOF {
				break
			}
			checkError(err)
		}

		fmt.Printf("Success %v bytes sent!\n", n)
	}
}