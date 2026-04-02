package agent

import (
	"github.com/titpetric/atkins/agent/greeting"
)

type (
	Greeter = greeting.Greeter
)

var (
	NewGreeter   = greeting.NewGreeter
	MatchFortune = greeting.MatchFortune
	Fortune      = greeting.Fortune
)
