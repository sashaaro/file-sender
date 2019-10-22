package main

import (
	"fmt"
	"github.com/pion/ice"
	"github.com/pion/sdp/v2"
	"strings"
)


type ICEProtocol int

const (
	// ICEProtocolUDP indicates the URL uses a UDP transport.
	ICEProtocolUDP ICEProtocol = iota + 1

	// ICEProtocolTCP indicates the URL uses a TCP transport.
	ICEProtocolTCP
)

// This is done this way because of a linter.
const (
	iceProtocolUDPStr = "udp"
	iceProtocolTCPStr = "tcp"
)

// NewICEProtocol takes a string and converts it to ICEProtocol
func NewICEProtocol(raw string) (ICEProtocol, error) {
	switch {
	case strings.EqualFold(iceProtocolUDPStr, raw):
		return ICEProtocolUDP, nil
	case strings.EqualFold(iceProtocolTCPStr, raw):
		return ICEProtocolTCP, nil
	default:
		return ICEProtocol(ice.Unknown), fmt.Errorf("unknown protocol: %s", raw)
	}
}

func (t ICEProtocol) String() string {
	switch t {
	case ICEProtocolUDP:
		return iceProtocolUDPStr
	case ICEProtocolTCP:
		return iceProtocolTCPStr
	default:
		return ice.ErrUnknownType.Error()
	}
}

// ICECandidateType represents the type of the ICE candidate used.
type ICECandidateType int

const (
	// ICECandidateTypeHost indicates that the candidate is of Host type as
	// described in https://tools.ietf.org/html/rfc8445#section-5.1.1.1. A
	// candidate obtained by binding to a specific port from an IP address on
	// the host. This includes IP addresses on physical interfaces and logical
	// ones, such as ones obtained through VPNs.
	ICECandidateTypeHost ICECandidateType = iota + 1

	// ICECandidateTypeSrflx indicates the the candidate is of Server
	// Reflexive type as described
	// https://tools.ietf.org/html/rfc8445#section-5.1.1.2. A candidate type
	// whose IP address and port are a binding allocated by a NAT for an ICE
	// agent after it sends a packet through the NAT to a server, such as a
	// STUN server.
	ICECandidateTypeSrflx

	// ICECandidateTypePrflx indicates that the candidate is of Peer
	// Reflexive type. A candidate type whose IP address and port are a binding
	// allocated by a NAT for an ICE agent after it sends a packet through the
	// NAT to its peer.
	ICECandidateTypePrflx

	// ICECandidateTypeRelay indicates the the candidate is of Relay type as
	// described in https://tools.ietf.org/html/rfc8445#section-5.1.1.2. A
	// candidate type obtained from a relay server, such as a TURN server.
	ICECandidateTypeRelay
)

// This is done this way because of a linter.
const (
	iceCandidateTypeHostStr  = "host"
	iceCandidateTypeSrflxStr = "srflx"
	iceCandidateTypePrflxStr = "prflx"
	iceCandidateTypeRelayStr = "relay"
)

// NewICECandidateType takes a string and converts it into ICECandidateType
func NewICECandidateType(raw string) (ICECandidateType, error) {
	switch raw {
	case iceCandidateTypeHostStr:
		return ICECandidateTypeHost, nil
	case iceCandidateTypeSrflxStr:
		return ICECandidateTypeSrflx, nil
	case iceCandidateTypePrflxStr:
		return ICECandidateTypePrflx, nil
	case iceCandidateTypeRelayStr:
		return ICECandidateTypeRelay, nil
	default:
		return ICECandidateType(ice.Unknown), fmt.Errorf("unknown ICE candidate type: %s", raw)
	}
}

func (t ICECandidateType) String() string {
	switch t {
	case ICECandidateTypeHost:
		return iceCandidateTypeHostStr
	case ICECandidateTypeSrflx:
		return iceCandidateTypeSrflxStr
	case ICECandidateTypePrflx:
		return iceCandidateTypePrflxStr
	case ICECandidateTypeRelay:
		return iceCandidateTypeRelayStr
	default:
		return ice.ErrUnknownType.Error()
	}
}

func getCandidateType(candidateType ice.CandidateType) (ICECandidateType, error) {
	switch candidateType {
	case ice.CandidateTypeHost:
		return ICECandidateTypeHost, nil
	case ice.CandidateTypeServerReflexive:
		return ICECandidateTypeSrflx, nil
	case ice.CandidateTypePeerReflexive:
		return ICECandidateTypePrflx, nil
	case ice.CandidateTypeRelay:
		return ICECandidateTypeRelay, nil
	default:
		// NOTE: this should never happen[tm]
		err := fmt.Errorf(
			"cannot convert ice.CandidateType into webrtc.ICECandidateType, invalid type: %s",
			candidateType.String())
		return ICECandidateType(ice.Unknown), err
	}
}

// ICECandidate represents a ice candidate
type ICECandidate struct {
	statsID        string
	Foundation     string           `json:"foundation"`
	Priority       uint32           `json:"priority"`
	Address        string           `json:"address"`
	Protocol       ICEProtocol      `json:"protocol"`
	Port           uint16           `json:"port"`
	Typ            ICECandidateType `json:"type"`
	Component      uint16           `json:"component"`
	RelatedAddress string           `json:"relatedAddress"`
	RelatedPort    uint16           `json:"relatedPort"`
}

// Conversion for package ice

func newICECandidatesFromICE(iceCandidates []ice.Candidate) ([]ICECandidate, error) {
	candidates := []ICECandidate{}

	for _, i := range iceCandidates {
		c, err := newICECandidateFromICE(i)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}

	return candidates, nil
}

func newICECandidateFromICE(i ice.Candidate) (ICECandidate, error) {
	typ, err := convertTypeFromICE(i.Type())
	if err != nil {
		return ICECandidate{}, err
	}
	protocol, err := NewICEProtocol(i.NetworkType().NetworkShort())
	if err != nil {
		return ICECandidate{}, err
	}

	c := ICECandidate{
		statsID:    i.ID(),
		Foundation: "foundation",
		Priority:   i.Priority(),
		Address:    i.Address(),
		Protocol:   protocol,
		Port:       uint16(i.Port()),
		Component:  i.Component(),
		Typ:        typ,
	}

	if i.RelatedAddress() != nil {
		c.RelatedAddress = i.RelatedAddress().Address
		c.RelatedPort = uint16(i.RelatedAddress().Port)
	}

	return c, nil
}

func (c ICECandidate) toICE() (ice.Candidate, error) {
	candidateID := c.statsID
	switch c.Typ {
	case ICECandidateTypeHost:
		config := ice.CandidateHostConfig{
			CandidateID: candidateID,
			Network:     c.Protocol.String(),
			Address:     c.Address,
			Port:        int(c.Port),
			Component:   c.Component,
		}
		return ice.NewCandidateHost(&config)
	case ICECandidateTypeSrflx:
		config := ice.CandidateServerReflexiveConfig{
			CandidateID: candidateID,
			Network:     c.Protocol.String(),
			Address:     c.Address,
			Port:        int(c.Port),
			Component:   c.Component,
			RelAddr:     c.RelatedAddress,
			RelPort:     int(c.RelatedPort),
		}
		return ice.NewCandidateServerReflexive(&config)
	case ICECandidateTypePrflx:
		config := ice.CandidatePeerReflexiveConfig{
			CandidateID: candidateID,
			Network:     c.Protocol.String(),
			Address:     c.Address,
			Port:        int(c.Port),
			Component:   c.Component,
			RelAddr:     c.RelatedAddress,
			RelPort:     int(c.RelatedPort),
		}
		return ice.NewCandidatePeerReflexive(&config)
	case ICECandidateTypeRelay:
		config := ice.CandidateRelayConfig{
			CandidateID: candidateID,
			Network:     c.Protocol.String(),
			Address:     c.Address,
			Port:        int(c.Port),
			Component:   c.Component,
			RelAddr:     c.RelatedAddress,
			RelPort:     int(c.RelatedPort),
		}
		return ice.NewCandidateRelay(&config)
	default:
		return nil, fmt.Errorf("unknown candidate type: %s", c.Typ)
	}
}

func convertTypeFromICE(t ice.CandidateType) (ICECandidateType, error) {
	switch t {
	case ice.CandidateTypeHost:
		return ICECandidateTypeHost, nil
	case ice.CandidateTypeServerReflexive:
		return ICECandidateTypeSrflx, nil
	case ice.CandidateTypePeerReflexive:
		return ICECandidateTypePrflx, nil
	case ice.CandidateTypeRelay:
		return ICECandidateTypeRelay, nil
	default:
		return ICECandidateType(t), fmt.Errorf("unknown ICE candidate type: %s", t)
	}
}

func (c ICECandidate) String() string {
	ic, err := c.toICE()
	if err != nil {
		return fmt.Sprintf("%#v failed to convert to ICE: %s", c, err)
	}
	return ic.String()
}

func iceCandidateToSDP(c ICECandidate) sdp.ICECandidate {
	return sdp.ICECandidate{
		Foundation:     c.Foundation,
		Priority:       c.Priority,
		Address:        c.Address,
		Protocol:       c.Protocol.String(),
		Port:           c.Port,
		Component:      c.Component,
		Typ:            c.Typ.String(),
		RelatedAddress: c.RelatedAddress,
		RelatedPort:    c.RelatedPort,
	}
}

// ICECandidateInit is used to serialize ice candidates
type ICECandidateInit struct {
	Candidate        string  `json:"candidate"`
	SDPMid           *string `json:"sdpMid,omitempty"`
	SDPMLineIndex    *uint16 `json:"sdpMLineIndex,omitempty"`
	UsernameFragment string  `json:"usernameFragment"`
}

// ToJSON returns an ICECandidateInit
// as indicated by the spec https://w3c.github.io/webrtc-pc/#dom-rtcicecandidate-tojson
func (c ICECandidate) ToJSON() ICECandidateInit {
	var sdpmLineIndex uint16
	return ICECandidateInit{
		Candidate:     fmt.Sprintf("candidate:%s", iceCandidateToSDP(c).Marshal()),
		SDPMLineIndex: &sdpmLineIndex,
	}
}
