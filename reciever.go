package main

import "github.com/samber/do"

type (
	SerialReciever struct {
	}

	RandomTestReciever struct {
	}
)

func NewSerialRecieverService(i *do.Injector) (*Reciever, error) {
	return &SerialReciever{}, nil
}

func NewRandomTestRecieverService(i *do.Injector) (*Reciever, error) {
	return &RandomTestReciever{}, nil
}
