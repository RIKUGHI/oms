package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rikughi/commons/discovery"
)

type Registry struct {
	sync.RWMutex
	serviceAddrs map[string]map[string]*serviceInstance
}

type serviceInstance struct {
	hostPort   string
	lastActive time.Time
}

// NewRegistry creates a new in-memory registry instance.
func NewRegistry() *Registry {
	return &Registry{
		serviceAddrs: make(map[string]map[string]*serviceInstance),
	}
}

// Register creates a service record in the registry
func (r *Registry) Register(ctx context.Context, instanceID string, serviceName string, hostPort string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.serviceAddrs[serviceName]; !ok {
		r.serviceAddrs[serviceName] = map[string]*serviceInstance{}
	}
	r.serviceAddrs[serviceName][instanceID] = &serviceInstance{
		hostPort:   hostPort,
		lastActive: time.Now(),
	}
	return nil
}

// Deregister removes a service record from the registry
func (r *Registry) Deregister(ctx context.Context, instanceID string, serviceName string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.serviceAddrs[serviceName]; !ok {
		return discovery.ErrNotFound
	}

	delete(r.serviceAddrs[serviceName], instanceID)
	return nil
}

// HealthCheck marks a service instance as active
func (r *Registry) HealthCheck(instanceID string, serviceName string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.serviceAddrs[serviceName]; !ok {
		return errors.New("service not registered yet")
	}

	if _, ok := r.serviceAddrs[serviceName][instanceID]; !ok {
		return errors.New("service instance not registered yet")
	}

	r.serviceAddrs[serviceName][instanceID].lastActive = time.Now()
	return nil
}

// Discover returns a list of service instances from the registry
func (r *Registry) Discover(ctx context.Context, serviceName string) ([]string, error) {
	r.RLock()
	defer r.RUnlock()

	if len(r.serviceAddrs[serviceName]) == 0 {
		return nil, discovery.ErrNotFound
	}
	var res []string

	for _, v := range r.serviceAddrs[serviceName] {
		if time.Since(v.lastActive) > 5*time.Second {
			continue
		}

		res = append(res, v.hostPort)
	}
	return res, nil
}
