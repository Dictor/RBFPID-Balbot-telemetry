package main

import (
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type ()

var (
	GlobalLogger = logrus.New()

	argSerialRecieverEnable, argRandtestRecieverEnable bool
	argSerialRecieverPort                              string
	argSerialRecieverBaud                              int
)

func main() {
	var (
		rootCmd = &cobra.Command{
			Use:   "telemetry <reciever> [reciever args...]",
			Short: "Telemetry reciever and UI for RBF-PID balbot",
			RunE:  execRoot,
		}
	)

	rootCmd.Flags().BoolVar(&argSerialRecieverEnable, "serial", false, "enable serial reciever")
	rootCmd.Flags().BoolVar(&argRandtestRecieverEnable, "rand", false, "enable random test reciever")
	rootCmd.MarkFlagsMutuallyExclusive("serial", "rand")
	rootCmd.MarkFlagsOneRequired("serial", "rand")

	rootCmd.Flags().StringVar(&argSerialRecieverPort, "port", "", "port name when using serial reciever")
	rootCmd.Flags().IntVar(&argSerialRecieverBaud, "baud", 9600, "baudrate when using serial reciever")
	rootCmd.MarkFlagsRequiredTogether("serial", "port")
	rootCmd.Execute()
}

func execRoot(cmd *cobra.Command, args []string) error {
	injector := do.New()

	if argRandtestRecieverEnable {
		do.Provide[Reciever](injector, NewRandomTestRecieverService)
	} else if argSerialRecieverEnable {
		do.Provide[Reciever](injector, NewSerialRecieverService)
	}

	recv := do.MustInvoke[Reciever](injector)
	var (
		sigChan  chan SignalMessage = make(chan SignalMessage, 10)
		statChan chan StatusMessage = make(chan StatusMessage, 10)
	)

	if err := recv.AssignChannel(statChan, sigChan); err != nil {
		GlobalLogger.WithError(err).Error("failed to assign message pipe")
		return err
	}

	if err := recv.Listen(); err != nil {
		GlobalLogger.WithError(err).Error("failed to assign message pipe")
		return err
	}

	RunGUI(sigChan, statChan)

	return nil
}
