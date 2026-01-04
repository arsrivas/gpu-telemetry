package mock

import "gpu-telemetry/model"

// StoreMock implements storage.Store
type StoreMock struct {
	GPUsFn      func() ([]string, error)
	GPUExistsFn func(id string) (bool, error)
	TelemetryFn func(id string, start, end *int64) ([]model.Telemetry, error)
	InsertFn    func(model.Telemetry) error
	PingFn      func() error
	CloseFn     func() error
}

func (m *StoreMock) GPUs() ([]string, error) {
	return m.GPUsFn()
}

func (m *StoreMock) GPUExists(id string) (bool, error) {
	return m.GPUExistsFn(id)
}

func (m *StoreMock) Telemetry(id string, start, end *int64) ([]model.Telemetry, error) {
	return m.TelemetryFn(id, start, end)
}

func (m *StoreMock) Insert(t model.Telemetry) error {
	if m.InsertFn != nil {
		return m.InsertFn(t)
	}
	return nil
}

func (m *StoreMock) Ping() error {
	return m.PingFn()
}

func (m *StoreMock) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}
