package main

import (
	"github.com/samber/do"
)

type (
	StatusMessage struct {
		Status       string
		StatusReason string
		KP, KI, KD   float32
		Error        float32
		Output       float32
	}

	SignalMessage struct {
		Signal  int
		Message string
	}

	Reciever interface {
		AssignChannel(chan<- StatusMessage, chan<- SignalMessage) error
		Listen() error
		do.Shutdownable
		do.Healthcheckable
	}
)

func main() {

}
