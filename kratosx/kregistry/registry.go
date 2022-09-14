package kregistry

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	_ registry.Registrar = &Registry{}
	_ registry.Discovery = &Registry{}
)

// Option is etcd registry option.
type Option func(o *options)

type options struct {
	ctx             context.Context
	dialTimeout     time.Duration
	ttl             time.Duration
	maxRetry        int
	useConfigHost   bool // 是否使用配置的host
	weight          int  // 权重, 可以为空
	useSystemWeight bool // 是否使用system(cpu, mem)算出来的权重
}

// Context with registry context.
func Context(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

// WithDialTimeout with connect dialTimeout.
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(o *options) { o.dialTimeout = dialTimeout }
}

// RegisterTTL with register ttl.
func RegisterTTL(ttl time.Duration) Option {
	return func(o *options) { o.ttl = ttl }
}

func MaxRetry(num int) Option {
	return func(o *options) { o.maxRetry = num }
}

// WithWeight with weight.
func WithWeight(weight int) Option {
	return func(o *options) { o.weight = weight }
}

// WithUseSystemWeight with useSystemWeight.
func WithUseSystemWeight(useSystemWeight bool) Option {
	return func(o *options) { o.useSystemWeight = useSystemWeight }
}

// Registry is etcd registry.
type Registry struct {
	id      uint32
	prefix  string
	service string
	Host    string
	Port    int
	opts    *options
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
}

// New creates etcd registry
func New(endpoints []string, prefix, service, host string, port int, opts ...Option) (r *Registry, err error) {
	op := &options{
		ctx:      context.Background(),
		ttl:      time.Second * 10,
		maxRetry: 5,
	}
	for _, o := range opts {
		o(op)
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: op.dialTimeout,
	})
	if err != nil {
		return nil, err
	}

	return &Registry{
		prefix:  prefix,
		service: service,
		Host:    host,
		Port:    port,
		opts:    op,
		client:  client,
		kv:      clientv3.NewKV(client),
	}, nil
}

// Register the registration.
func (r *Registry) Register(ctx context.Context, service *registry.ServiceInstance) error {
	// get id
	parseUint, err := strconv.ParseUint(service.ID, 10, 32)
	if err != nil {
		return err
	}
	r.id = uint32(parseUint)
	key := r.getKey()
	value, err := marshal(r, service)
	if err != nil {
		return err
	}
	if r.lease != nil {
		r.lease.Close()
	}
	r.lease = clientv3.NewLease(r.client)
	leaseID, err := r.registerWithKV(ctx, key, value)
	if err != nil {
		return err
	}

	go r.heartBeat(r.opts.ctx, leaseID, key, value)
	return nil
}

// Deregister the registration.
func (r *Registry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	defer func() {
		if r.lease != nil {
			r.lease.Close()
		}
	}()
	_, err := r.client.Delete(ctx, r.getKey())
	return err
}

// GetService return the service instances in memory according to the service name.
func (r *Registry) GetService(ctx context.Context, name string) ([]*registry.ServiceInstance, error) {
	resp, err := r.kv.Get(ctx, name, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	items := make([]*registry.ServiceInstance, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		si, err := unmarshal(kv.Value)
		if err != nil {
			return nil, err
		}
		items = append(items, si)
	}
	return items, nil
}

// Watch creates a watcher according to the service name.
func (r *Registry) Watch(ctx context.Context, name string) (registry.Watcher, error) {
	return newWatcher(ctx, name, r.client)
}

func (r *Registry) getKey() string {
	return fmt.Sprintf("%s%s/%d", r.prefix, r.service, r.id)
}

// registerWithKV create a new lease, return current leaseID
func (r *Registry) registerWithKV(ctx context.Context, key string, value string) (clientv3.LeaseID, error) {
	grant, err := r.lease.Grant(ctx, int64(r.opts.ttl.Seconds()))
	if err != nil {
		return 0, err
	}
	_, err = r.client.Put(ctx, key, value, clientv3.WithLease(grant.ID))
	if err != nil {
		return 0, err
	}
	return grant.ID, nil
}

func (r *Registry) heartBeat(ctx context.Context, leaseID clientv3.LeaseID, key string, value string) {
	curLeaseID := leaseID
	kac, err := r.client.KeepAlive(ctx, leaseID)
	if err != nil {
		curLeaseID = 0
	}
	rand.Seed(time.Now().Unix())

	for {
		if curLeaseID == 0 {
			// try to registerWithKV
			retreat := []int{}
			for retryCnt := 0; retryCnt < r.opts.maxRetry; retryCnt++ {
				if ctx.Err() != nil {
					return
				}
				// prevent infinite blocking
				idChan := make(chan clientv3.LeaseID, 1)
				errChan := make(chan error, 1)
				cancelCtx, cancel := context.WithCancel(ctx)
				go func() {
					defer cancel()
					id, registerErr := r.registerWithKV(cancelCtx, key, value)
					if registerErr != nil {
						errChan <- registerErr
					} else {
						idChan <- id
					}
				}()

				select {
				case <-time.After(3 * time.Second):
					cancel()
					continue
				case <-errChan:
					continue
				case curLeaseID = <-idChan:
				}

				kac, err = r.client.KeepAlive(ctx, curLeaseID)
				if err == nil {
					break
				}
				retreat = append(retreat, 1<<retryCnt)
				time.Sleep(time.Duration(retreat[rand.Intn(len(retreat))]) * time.Second)
			}
			if _, ok := <-kac; !ok {
				// retry failed
				continue
			}
		}

		select {
		case _, ok := <-kac:
			if !ok {
				if ctx.Err() != nil {
					// channel closed due to context cancel
					return
				}
				// need to retry registration
				curLeaseID = 0
				continue
			}
		case <-r.opts.ctx.Done():
			return
		}
	}
}
