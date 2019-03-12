package svc

import "os"

type mockProgram struct {
	start  func() error
	stop   func() error
	init   func(Environment) error
	notify func(os.Signal) error
}

func (p *mockProgram) Start() error {
	return p.start()
}

func (p *mockProgram) Stop() error {
	return p.stop()
}

func (p *mockProgram) Init(wse Environment) error {
	return p.init(wse)
}

func (p *mockProgram) Notify(sig os.Signal) error {
	return p.notify(sig)
}

func makeProgram(startCalled, stopCalled, initCalled, notifyCalled *int) *mockProgram {
	return &mockProgram{
		start: func() error {
			*startCalled++
			return nil
		},
		stop: func() error {
			*stopCalled++
			return nil
		},
		init: func(wse Environment) error {
			*initCalled++
			return nil
		},
		notify: func(signal os.Signal) error {
			*notifyCalled++
			return nil
		},
	}
}
