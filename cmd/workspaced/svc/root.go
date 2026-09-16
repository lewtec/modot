package svc

import ()

type Command struct {
	Osmardetector *Osmardetector
	ReniceHungry  *ReniceHungry `cmd:"renice-hungry"`
	Screencaps    *Screencaps
	Vncd          *Vncd
}

func (Command) Description() string {
	return "Background services"
}
