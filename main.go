package main

import (
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
	}

	agent, err := ice.NewAgent(config)
	checkError(err)
	err = agent.OnCandidate(func(candidate ice.Candidate) {
		if candidate == nil {

		} else {
			fmt.Printf("%s\n\n", candidate.String())
		}
	})
	checkError(err)

	err = agent.OnConnectionStateChange(func(state ice.ConnectionState) {
		fmt.Printf("%s\n", state.String())
	})
	checkError(err)


	err = agent.GatherCandidates()
	checkError(err)

	candidates, err := agent.GetLocalCandidates()
	checkError(err)

	fmt.Printf("%v", candidates)
	//err = agent.AddRemoteCandidate(candidate)


	select {}
}