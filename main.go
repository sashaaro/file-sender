package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pion/ice"
	"os"
	"context"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	url, err := ice.ParseURL("stun:stun.l.google.com:19302")
	checkError(err)
	config := &ice.AgentConfig{
		Urls: []*ice.URL{url},
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
	err = agent.OnCandidate(func(candidate ice.Candidate) {
		if candidate == nil {
			fmt.Printf("No candidate\n\n")
		} else {
			fmt.Printf("%s\n\n", candidate.String())
		}
	})
	checkError(err)

	err = agent.OnConnectionStateChange(func(state ice.ConnectionState) {
		fmt.Printf("State Change: %s\n", state.String())
	})
	checkError(err)

	/*err = agent.GatherCandidates()
	checkError(err)*/

	candidates, err := agent.GetLocalCandidates()
	iceCandidates, err := newICECandidatesFromICE(candidates)
	checkError(err)

	content, err := json.Marshal(iceCandidates)
	checkError(err)

	fmt.Printf("%s\n", content)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	response := scanner.Text()

	var partnerCandidates []ICECandidate
	err = json.Unmarshal([]byte(response), &partnerCandidates)
	checkError(err)

	uflag, pass := agent.GetLocalUserCredentials()
	fmt.Printf("uflag %s pass %s\n", uflag, pass)

	scanner.Scan()
	partnerUflag := scanner.Text()
	scanner.Scan()
	partnerPass := scanner.Text()


	for _, c := range partnerCandidates {
		i, err := c.toICE()
		checkError(err)
		err = agent.AddRemoteCandidate(i)
		checkError(err)
	}
	scanner.Scan()
	mode := scanner.Text()

	var conn *ice.Conn
	if mode != "" {
		conn, err = agent.Accept(context.Background(), partnerUflag, partnerPass)
		conn.Write([]byte("Hello"))
	} else {
		conn, err = agent.Dial(context.Background(), partnerUflag, partnerPass)
		conn.Write([]byte("world"))
	}
	checkError(err)

	for {
		buffer := make([]byte, 100)
		n, err := conn.Read(buffer)
		checkError(err)
		fmt.Printf("\nRead %v bytes\n", n)
		fmt.Printf(string(buffer))
	}
}