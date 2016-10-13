package upstream

type Upstream struct {
	ServiceName   string `json:"ServiceName"`
	FrontendPort  string
	FrontendIp    string
	FrontendProto string

	Targets []*Target `json:"Target"`
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
