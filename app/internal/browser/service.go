package browser

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/Seaony/Momento/app/internal/i18n"
)

type State string

const (
	Disabled     State = "disabled"
	Running      State = "running"
	PortOccupied State = "port_occupied"
	Failed       State = "failed"
)

type Status struct {
	State   State
	Address string
	Err     error
}

type Service struct {
	ctx      context.Context
	address  string
	handler  http.Handler
	onChange func(Status)
	opMu     sync.Mutex
	mu       sync.RWMutex
	status   Status
	run      *serviceRun
}

type serviceRun struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func NewService(ctx context.Context, handler http.Handler, onChange func(Status)) *Service {
	return newService(ctx, Address, handler, onChange)
}

func newService(ctx context.Context, address string, handler http.Handler, onChange func(Status)) *Service {
	return &Service{ctx: ctx, address: address, handler: handler, onChange: onChange, status: Status{State: Disabled, Address: address}}
}

func (s *Service) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// SetEnabled 会等待旧服务停止，调用方应在后台执行。
func (s *Service) SetEnabled(enabled bool) error {
	s.opMu.Lock()
	status, changed, err := s.setEnabled(enabled)
	s.opMu.Unlock()
	if changed {
		s.notify(status)
	}
	return err
}

func (s *Service) setEnabled(enabled bool) (Status, bool, error) {
	if !enabled {
		if s.run != nil {
			run := s.run
			s.run = nil
			run.cancel()
			<-run.done
			slog.Info("浏览器扩展服务已关闭", "address", s.address)
		}
		if s.Status().State == Disabled {
			return s.Status(), false, nil
		}
		status := Status{State: Disabled, Address: s.address}
		s.setStatus(status)
		return status, true, nil
	}
	if err := s.ctx.Err(); err != nil {
		return s.Status(), false, err
	}
	if s.run != nil {
		return s.Status(), false, nil
	}
	listener, err := net.Listen("tcp4", s.address)
	if err != nil {
		state := Failed
		if isAddressInUse(err) {
			state = PortOccupied
		}
		err = i18n.WrapError(err, "error_browser_start")
		status := Status{State: state, Address: s.address, Err: err}
		s.setStatus(status)
		slog.Error("浏览器扩展服务启动失败", "address", s.address, "error", err)
		return status, true, err
	}
	ctx, cancel := context.WithCancel(s.ctx)
	run := &serviceRun{cancel: cancel, done: make(chan struct{})}
	s.run = run
	status := Status{State: Running, Address: listener.Addr().String()}
	s.setStatus(status)
	slog.Info("浏览器扩展服务已启动", "address", status.Address)
	go s.serve(ctx, listener, run)
	return status, true, nil
}

func (s *Service) serve(ctx context.Context, listener net.Listener, run *serviceRun) {
	err := serve(ctx, listener, s.handler)
	run.cancel()
	close(run.done)
	s.opMu.Lock()
	if s.run != run {
		s.opMu.Unlock()
		return
	}
	s.run = nil
	status := Status{State: Disabled, Address: s.address}
	if err != nil {
		status.State = Failed
		status.Err = i18n.WrapError(err, "error_browser_start")
		slog.Error("浏览器扩展服务意外停止", "address", s.address, "error", err)
	} else {
		slog.Info("浏览器扩展服务已关闭", "address", s.address)
	}
	s.setStatus(status)
	s.opMu.Unlock()
	s.notify(status)
}

func (s *Service) setStatus(status Status) {
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()
}

func (s *Service) notify(status Status) {
	if s.onChange != nil {
		s.onChange(status)
	}
}
