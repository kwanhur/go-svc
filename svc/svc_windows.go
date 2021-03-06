// +build windows

package svc

import (
	"os"
	"path/filepath"
	"sync"
	"syscall"

	wsvc "golang.org/x/sys/windows/svc"
)

// Create variables for svc and signal functions so we can mock them in tests
var svcIsAnInteractiveSession = wsvc.IsAnInteractiveSession
var svcRun = wsvc.Run

type windowsService struct {
	i             Service
	errSync       sync.Mutex
	stopStartErr  error
	isInteractive bool
	signals       []os.Signal
	signalMap     map[os.Signal]struct{}
	Name          string
}

// Run runs an implementation of the Service interface.
//
// Run will block until the Windows Service is stopped or Ctrl+C is pressed if
// running from the console.
//
// Stopping the Windows Service and Ctrl+C will call the Service's Stop method to
// initiate a graceful shutdown.
//
// Note that WM_CLOSE is not handled (end task) and the Service's Stop method will
// not be called.
//
// The sig parameter is to keep parity with the non-Windows API. Only syscall.SIGINT
// (Ctrl+C) can be handled on Windows. Nevertheless, you can override the default
// signals which are handled by specifying sig.
func Run(service Service, sig ...os.Signal) error {
	var err error

	interactive, err := svcIsAnInteractiveSession()
	if err != nil {
		return err
	}

	if len(sig) == 0 {
		sig = []os.Signal{syscall.SIGINT}
	}

	ws := &windowsService{
		i:             service,
		isInteractive: interactive,
		signals:       sig,
		signalMap:     make(map[os.Signal]struct{}),
	}

	if ws.IsWindowsService() {
		// the working directory for a Windows Service is C:\Windows\System32
		// this is almost certainly not what the user wants.
		dir := filepath.Dir(os.Args[0])
		if err = os.Chdir(dir); err != nil {
			return err
		}
	}

	if err = service.Init(ws); err != nil {
		return err
	}

	return ws.run()
}

func (ws *windowsService) setError(err error) {
	ws.errSync.Lock()
	ws.stopStartErr = err
	ws.errSync.Unlock()
}

func (ws *windowsService) getError() error {
	ws.errSync.Lock()
	err := ws.stopStartErr
	ws.errSync.Unlock()
	return err
}

func (ws *windowsService) IsWindowsService() bool {
	return !ws.isInteractive
}

func (ws *windowsService) run() error {
	ws.setError(nil)
	if ws.IsWindowsService() {
		// Return error messages from start and stop routines
		// that get executed in the Execute method.
		// Guarded with a mutex as it may run a different thread
		// (callback from Windows).
		runErr := svcRun(ws.Name, ws)
		startStopErr := ws.getError()
		if startStopErr != nil {
			return startStopErr
		}
		if runErr != nil {
			return runErr
		}
		return nil
	}

	err := ws.i.Start()
	if err != nil {
		return err
	}

	for sig := range signalNotifier {
		ws.signals = append(ws.signals, sig)
	}
	for _, sig := range ws.signals{
		ws.signalMap[sig] = struct{}{}
	}

	signalChan := make(chan os.Signal, 1)
	signalNotify(signalChan, ws.signals...)

	for {
		sig := <-signalChan
		if notify, ok := signalNotifier[sig]; ok {
			notify(sig)
		}else if _, ok := ws.signalMap[sig]; ok {
			return ws.i.Stop()
		}
	}
}

// Execute is invoked by Windows
func (ws *windowsService) Execute(args []string, r <-chan wsvc.ChangeRequest, changes chan<- wsvc.Status) (bool, uint32) {
	const cmdsAccepted = wsvc.AcceptStop | wsvc.AcceptShutdown
	changes <- wsvc.Status{State: wsvc.StartPending}

	if err := ws.i.Start(); err != nil {
		ws.setError(err)
		return true, 1
	}

	changes <- wsvc.Status{State: wsvc.Running, Accepts: cmdsAccepted}
loop:
	for {
		c := <-r
		switch c.Cmd {
		case wsvc.Interrogate:
			changes <- c.CurrentStatus
		case wsvc.Stop, wsvc.Shutdown:
			changes <- wsvc.Status{State: wsvc.StopPending}
			err := ws.i.Stop()
			if err != nil {
				ws.setError(err)
				return true, 2
			}
			break loop
		default:
			continue loop
		}
	}

	return false, 0
}
