package svc

import (
	"os"
	"os/signal"
	"syscall"
	"errors"
)

// Create variable signal.Notify function so we can mock it in tests
var signalNotify = signal.Notify
var signalNotifier map[os.Signal]SignalReceive

// Service interface contains Start and Stop methods which are called
// when the service is started and stopped. The Init method is called
// before the service is started, and after it's determined if the program
// is running as a Windows Service.
//
// The Start and Init methods must be non-blocking.
//
// Implement this interface and pass it to the Run function to start your program.
type Service interface {
	// Init is called before the program/service is started and after it's
	// determined if the program is running as a Windows Service. This method must
	// be non-blocking.
	Init(Environment) error

	// Start is called after Init. This method must be non-blocking.
	Start() error

	// Stop is called in response to syscall.SIGINT, syscall.SIGTERM, or when a
	// Windows Service is stopped.
	Stop() error
}

// Environment contains information about the environment
// your application is running in.
type Environment interface {
	// IsWindowsService reports whether the program is running as a Windows Service.
	IsWindowsService() bool
}

// SignalReceive signal information to process
// implement case by case
type SignalReceive func(signal os.Signal)

// Notify register signal to receive
// specify receive func implement by yourself
func Notify(sig os.Signal, receive SignalReceive)error  {
	switch sig {
	case syscall.SIGINT,syscall.SIGTERM:
		return errors.New("signal was reserved")
	default:
		if signalNotifier == nil{
			signalNotifier = make(map[os.Signal]SignalReceive)
		}

		signalNotifier[sig] = receive
	}

	return nil
}
