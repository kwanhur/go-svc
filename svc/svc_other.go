// +build !windows

package svc

import (
	"os"
	"syscall"
)

type otherService struct {
	service Service
	signals []os.Signal
	signalMap map[os.Signal]struct{}
}

// Run runs your Service.
//
// Run will block until one of the signals specified in sig is received.
// If sig is empty syscall.SIGINT and syscall.SIGTERM are used by default.
func Run(service Service, sig ...os.Signal) error {
	env := environment{}
	if err := service.Init(env); err != nil {
		return err
	}

	if err := service.Start(); err != nil {
		return err
	}

	if len(sig) == 0 {
		sig = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	svr := &otherService{}
	svr.signals = sig

	return svr.run()
}

func (svr *otherService) run()error  {
	for sig := range signalNotifier {
		svr.signals = append(svr.signals, sig)
	}
	for _, sig := range svr.signals {
		svr.signalMap[sig] = struct{}{}
	}

	signalChan := make(chan os.Signal, 1)
	signalNotify(signalChan, svr.signals...)

	for {
		sig := <-signalChan
		if notify, ok := signalNotifier[sig]; ok {
			notify(sig)
		}else if _, ok := svr.signalMap[sig]; ok {
			return svr.service.Stop()
		}
	}
}

type environment struct{}

func (environment) IsWindowsService() bool {
	return false
}
