package main

import (
	"bufio"
	"context"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/samber/do"
	"github.com/samber/lo"
	"go.bug.st/serial"
)

type (
	StatusMessage struct {
		Time         float32
		Status       string
		StatusReason string
		KP, KI, KD   float32
		Error        float32
		Output       float32
	}

	SignalMessage struct {
		Time    float32
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
		statChan  chan<- StatusMessage
		sigChan   chan<- SignalMessage
		ctx       context.Context
		ctxCancel context.CancelFunc
		port      serial.Port
	}

	RandomTestReciever struct {
		statChan  chan<- StatusMessage
		sigChan   chan<- SignalMessage
		ctx       context.Context
		ctxCancel context.CancelFunc
	}
)

func (recv *SerialReciever) OpenPort(portName string, portBaud int) error {
	cfg := &serial.Mode{
		BaudRate: portBaud,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	var err error
	recv.port, err = serial.Open(portName, cfg)
	return err
}

func (recv *SerialReciever) AssignChannel(stat chan<- StatusMessage, sig chan<- SignalMessage) error {
	recv.statChan = stat
	recv.sigChan = sig
	return nil
}

func (recv *SerialReciever) Listen() error {
	recv.ctx, recv.ctxCancel = context.WithCancel(context.Background())
	r, w := io.Pipe()
	copyBuffer := make([]byte, 32768)
	recv.port.SetReadTimeout(serial.NoTimeout)

	go func() {
		for {
			select {
			case <-recv.ctx.Done():
				return
			default:
			}
			n, err := recv.port.Read(copyBuffer)
			if err != nil {
				GlobalLogger.WithError(err).Error("failed to copy serial buffer")
			}
			if n == 0 {
				time.Sleep(100 * time.Millisecond)
			} else {
				if nn, err := w.Write(copyBuffer[:n]); err != nil {
					GlobalLogger.WithError(err).Errorf("buffer is fragmented, expected writing %d, wrote %d", len(copyBuffer), nn)
				}
			}
		}
	}()

	go func() {
		nr := bufio.NewReader(r)
		for {
			select {
			case <-recv.ctx.Done():
				recv.sigChan <- SignalMessage{
					Signal:  -1,
					Message: "reciever halted",
				}
				return
			default:
				time.Sleep(10 * time.Millisecond)
			}

			rawLine, isPrefix, err := nr.ReadLine()
			if isPrefix {
				continue
			}
			if err != nil {
				GlobalLogger.WithError(err).Errorf("failed to read buffer, isPrefix=%t", isPrefix)
				continue
			}

			tokens := strings.Split(string(rawLine), ",")
			if len(tokens) != 6 {
				GlobalLogger.Errorf("token is incomplete: %d tokens", len(tokens))
				continue
			}

			values := lo.Map[string, float32](tokens, func(item string, index int) float32 {
				f, err := strconv.ParseFloat(item, 32)
				if err != nil {
					GlobalLogger.WithError(err).Errorf("failed to parse serial token : %s", item)
					f = 0
				}
				return float32(f)
			})

			recv.statChan <- StatusMessage{
				Status:       "normal",
				StatusReason: "",
				KP:           values[3],
				KI:           values[4],
				KD:           values[5],
				Error:        values[1],
				Output:       values[2],
				Time:         values[0],
			}
		}
	}()
	return nil
}

func (recv *SerialReciever) Shutdown() error {
	if err := recv.port.Close(); err != nil {
		GlobalLogger.WithError(err).Error("failed to close serial port")
	}
	recv.ctxCancel()
	return nil
}

func (recv *SerialReciever) HealthCheck() error {
	return nil
}

func (recv *RandomTestReciever) AssignChannel(stat chan<- StatusMessage, sig chan<- SignalMessage) error {
	recv.statChan = stat
	recv.sigChan = sig
	return nil
}

func (recv *RandomTestReciever) Listen() error {
	recv.ctx, recv.ctxCancel = context.WithCancel(context.Background())

	go func() {
		timeStart := time.Now()
		for {
			timeCurrect := time.Now()
			secondElasped := float32(timeCurrect.Sub(timeStart).Seconds())
			recv.statChan <- StatusMessage{
				Status:       "normal",
				StatusReason: "random test reciever",
				KP:           rand.Float32(),
				KI:           rand.Float32(),
				KD:           rand.Float32(),
				Error:        rand.Float32(),
				Output:       rand.Float32(),
				Time:         secondElasped,
			}
			recv.sigChan <- SignalMessage{
				Signal:  0,
				Message: "reciever still alive",
				Time:    secondElasped,
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

			time.Sleep(200 * time.Millisecond)
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
	return &SerialReciever{}, nil
}

func NewRandomTestRecieverService(i *do.Injector) (Reciever, error) {
	return &RandomTestReciever{}, nil
}
