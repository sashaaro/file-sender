package main

import (
	"context"
	"fmt"
	"github.com/pion/ice"
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
	checkError(err)

	for _, c := range candidates {
		err = agent.AddRemoteCandidate(c)
		checkError(err)
	}

	fmt.Printf("%v\n", candidates)
	//err = agent.AddRemoteCandidate(candidate)

	uflag, pass := agent.GetLocalUserCredentials()
	fmt.Printf("uflag %s pass %s\n", uflag, pass)

	checkError(err)

	for {
		conn, err := agent.Accept(context.Background(), uflag, pass)
		checkError(err)

		buffer := make([]byte, 100)
		n, err := conn.Read(buffer)
		checkError(err);
		fmt.Printf("Read %v bytes", n)
	}
}