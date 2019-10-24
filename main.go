package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/pion/ice"
	//"github.com/pion/logging"
	//"github.com/pion/sctp"
	"io"
	"net/url"
	"os"
	"strings"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if os.Args[1] == "server" {
		server()
		return
	}

	u := url.URL{Scheme: "ws", Host: fmt.Sprintf("%s:8443", os.Args[1]), Path: "/ws"}

	var mode string
	if len(os.Args) > 2 {
		mode = os.Args[2]
	}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	checkError(err)

	stunUrl, err := ice.ParseURL("stun:stun.l.google.com:19302")
	checkError(err)
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
		// LoggerFactory: logging.NewDefaultLoggerFactory(),
	}

	agent, err := ice.NewAgent(config)
	checkError(err)

	err = agent.OnConnectionStateChange(func(state ice.ConnectionState) {
		fmt.Printf("State Change: %s\n", state.String())

		if state == ice.ConnectionStateConnected {

		}
	})

	go func() {
		bufio.NewReader(os.Stdin).ReadBytes('\n')

		candidates, err := agent.GetLocalCandidates()
		iceCandidates, err := newICECandidatesFromICE(candidates)
		checkError(err)

		content, err := json.Marshal(iceCandidates)
		checkError(err)

		err = c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("candidate|%s", string(content))))
		checkError(err)

		uflag, pass := agent.GetLocalUserCredentials()

		bufio.NewReader(os.Stdin).ReadBytes('\n')

		err = c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("credentials|%s|%s", uflag, pass)))
		checkError(err)
	}()


	var conn *ice.Conn

	for {
		_, res, err := c.ReadMessage()
		checkError(err)

		parts := strings.SplitN(string(res), "|", 2)

		fmt.Printf("Recieve %s\n", parts[0])
		switch parts[0] {
		case "candidate":
			var partnerCandidates []ICECandidate
			err := json.Unmarshal([]byte(parts[1]), &partnerCandidates)
			checkError(err)
			for _, c := range partnerCandidates {
				i, err := c.toICE()
				checkError(err)
				err = agent.AddRemoteCandidate(i)
				checkError(err)
			}
		case "credentials":
			credentials := strings.SplitN(string(parts[1]), "|", 2)

			if mode != "" {
				conn, err = agent.Accept(context.Background(), credentials[0], credentials[1])
				checkError(err)
				//_, err = conn.Write([]byte("Hello"))
				//checkError(err)
			} else {
				conn, err = agent.Dial(context.Background(), credentials[0], credentials[1])
				checkError(err)
				//_, err = conn.Write([]byte("world"))
				//checkError(err)
			}

			/*association, err := sctp.Client(sctp.Config{
				NetConn: conn,
				LoggerFactory: logging.NewDefaultLoggerFactory(),
			})
			checkError(err)
*/
			if mode != "" {
				// conn.Write([]byte("Hello"))

				//stream, err := association.AcceptStream()
				//checkError(err)
				f, err := os.Create("/tmp/PhpStorm-2019.1.1-copy.pdf")
				checkError(err)

				//defer stream.Close()
				defer  f.Close()

				//time.Sleep(2 * time.Second)
				n, err := io.Copy(f, conn)
				checkError(err)

				fmt.Printf("\nRecieve %v bytes\n", n)

			} else {
				// conn.Write([]byte("world"))

				//stream, err := association.OpenStream(777, sctp.PayloadTypeWebRTCBinary)
				//checkError(err)

				//f, err := os.Open("/Users/sasha/Downloads/PhpStorm-2019.1.1.dmg")
				//f, err := os.Open("/Users/sasha/Downloads/abc-1.jpg")
				f, err := os.Open("/Users/sasha/Downloads/The Social Smart Contract.pdf")
				checkError(err)


				//defer stream.Close()
				defer  f.Close()

				buffer := make([]byte, 50)//1024 * 2)
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

				//n, err := io.Copy(conn, f)
				//checkError(err)

				fmt.Printf("\nSend %v bytes\n", n)

			}

			/*for {
				buffer := make([]byte, 100)
				n, err := conn.Read(buffer)
				checkError(err)
				fmt.Printf("\nRead %v bytes\n", n)
				fmt.Printf(string(buffer))
			}*/
		}
	}
}