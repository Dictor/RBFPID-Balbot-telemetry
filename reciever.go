package main

import (
	"context"
	"math/rand"
	"time"

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

	SerialReciever struct {
	}

	RandomTestReciever struct {
		statChan  chan<- StatusMessage
		sigChan   chan<- SignalMessage
		ctx       context.Context
		ctxCancel context.CancelFunc
	}
)

func (recv *RandomTestReciever) AssignChannel(stat chan<- StatusMessage, sig chan<- SignalMessage) error {
	recv.statChan = stat
	recv.sigChan = sig
	return nil
}

func (recv *RandomTestReciever) Listen() error {
	recv.ctx, recv.ctxCancel = context.WithCancel(context.Background())

	go func() {
		for {
			recv.statChan <- StatusMessage{
				Status:       "normal",
				StatusReason: "random test reciever",
				KP:           rand.Float32(),
				KI:           rand.Float32(),
				KD:           rand.Float32(),
				Error:        rand.Float32(),
				Output:       rand.Float32(),
			}
			recv.sigChan <- SignalMessage{
				Signal:  0,
				Message: "reciever still alive",
			}

			select {
			case <-recv.ctx.Done():
				recv.sigChan <- SignalMessage{
					Signal:  -1,
					Message: "reciever halted",
				}
				return
			default:
			}

			time.Sleep(time.Second)
		}
	}()
	return nil
}

func (recv *RandomTestReciever) Shutdown() error {
	recv.ctxCancel()
	return nil
}

func (recv *RandomTestReciever) HealthCheck() error {
	return nil
}

func NewSerialRecieverService(i *do.Injector) (Reciever, error) {
	GlobalLogger.Panic("serial reciever isn't implemented yet")
	return nil, nil
	//return &SerialReciever{}, nil
}

func NewRandomTestRecieverService(i *do.Injector) (Reciever, error) {
	return &RandomTestReciever{}, nil
}
