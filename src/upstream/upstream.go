package upstream

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
)

type Upstream struct {
	State     *UpstreamState // new|listening|outdated|changed
	StaleMark bool           // mark if the current upstream not inuse anymore

	ServiceName   string `json:"ServiceName"`
	FrontendPort  string // port listen
	FrontendIp    string // ip listen
	FrontendProto string // http|https|tcp

	Targets []*Target `json:"Target"`
}

type UpstreamKey struct {
	Proto string
	Ip    string
	Port  string
}

type UpstreamStateEnum string

const (
	STATE_NEW       UpstreamStateEnum = "new"
	STATE_LISTENING UpstreamStateEnum = "listening"
	STATE_OUTDATED  UpstreamStateEnum = "outdated"
	STATE_CHANGED   UpstreamStateEnum = "changed"
)

type UpstreamState struct {
	state    UpstreamStateEnum
	upstream *Upstream
}

func NewUpstreamState(u *Upstream, newState UpstreamStateEnum) *UpstreamState {
	return &UpstreamState{
		upstream: u,
		state:    newState,
	}
}

func (us UpstreamState) Update(newState UpstreamStateEnum) {
	if us.state != newState {
		log.Debugf("change state of upstream <%s> to [%s]", us.upstream.ToString(), newState)
		us.state = newState
	}
}

func (u *Upstream) ToString() string {
	return fmt.Sprintf("%s-%s-%s-%s", u.ServiceName, u.FrontendProto, u.FrontendIp, u.FrontendPort)
}

func (u *Upstream) Equal(u1 *Upstream) bool {
	fieldsEqual := u.ServiceName == u1.ServiceName &&
		u.FrontendPort == u1.FrontendPort &&
		u.FrontendIp == u1.FrontendIp &&
		u.FrontendProto == u1.FrontendProto

	targetsSizeEqual := len(u.Targets) == len(u1.Targets)

	targetsEqual := true
	uTargets := make([]string, 0)
	for _, t := range u.Targets {
		uTargets = append(uTargets, t.ToString())
	}
	u1Targets := make([]string, 0)
	for _, t := range u1.Targets {
		u1Targets = append(u1Targets, t.ToString())
	}
	for index, targetStr := range uTargets {
		if targetStr != uTargets[index] {
			targetsEqual = false
		}
	}

	return fieldsEqual && targetsSizeEqual && targetsEqual
}

func (u *Upstream) FieldsEqualButTargetsDiffer(u1 *Upstream) bool {
	fieldsEqual := u.ServiceName == u1.ServiceName &&
		u.FrontendPort == u1.FrontendPort &&
		u.FrontendIp == u1.FrontendIp &&
		u.FrontendProto == u1.FrontendProto

	targetsSizeEqual := len(u.Targets) == len(u1.Targets)

	targetsEqual := true
	uTargets := make([]string, 0)
	for _, t := range u.Targets {
		uTargets = append(uTargets, t.ToString())
	}
	u1Targets := make([]string, 0)
	for _, t := range u1.Targets {
		u1Targets = append(u1Targets, t.ToString())
	}
	for index, targetStr := range uTargets {
		if targetStr != uTargets[index] {
			targetsEqual = false
		}
	}

	return (fieldsEqual && !targetsSizeEqual) || (fieldsEqual && !targetsEqual)
}

func (u *Upstream) FieldsEqual(u1 *Upstream) bool {
	fieldsEqual := u.ServiceName == u1.ServiceName &&
		u.FrontendPort == u1.FrontendPort &&
		u.FrontendIp == u1.FrontendIp &&
		u.FrontendProto == u1.FrontendProto

	return fieldsEqual
}

func (u *Upstream) SetState(newState UpstreamStateEnum) {
	if u.State == nil {
		u.State = NewUpstreamState(u, newState)
	} else {
		u.State.Update(newState)
	}
}

func (u *Upstream) StateIs(expectState UpstreamStateEnum) bool {
	return u.State.state == expectState
}

func (u *Upstream) Key() UpstreamKey {
	return UpstreamKey{Proto: u.FrontendProto, Ip: u.FrontendIp, Port: u.FrontendPort}
}
