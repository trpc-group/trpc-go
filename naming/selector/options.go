package selector

import (
	"context"

	"trpc.group/trpc-go/trpc-go/naming/circuitbreaker"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
)

var (
	defaultDiscoveryOptionsSize     = 2
	defaultServiceRouterOptionsSize = 2
	defaultLoadBalanceOptionsSize   = 2
)

// Options defines the call options.
type Options struct {
	// Ctx is the corresponding context to request.
	Ctx context.Context
	// Key is the hash key of stateful routing.
	Key string
	// Replicas is the replicas of a single node for stateful routing. It's optional, and used to
	// address hash ring.
	Replicas int
	// EnvKey is the environment key.
	EnvKey string
	// Namespace is the callee namespace.
	Namespace string
	// SourceNamespace is the caller namespace.
	SourceNamespace string
	// SourceServiceName is the caller service name.
	SourceServiceName string
	// SourceEnvName is the caller environment name.
	SourceEnvName string
	// SourceSetName if the caller set group.
	SourceSetName string
	// SourceMetadata is the caller metadata used to match routing.
	SourceMetadata map[string]string
	// DestinationEnvName is the callee environment name which is used to get node in the specific
	// environment.
	DestinationEnvName string
	// DestinationSetName is callee set name.
	DestinationSetName string
	// DestinationMetadata is the callee metadata used to match routing.
	DestinationMetadata map[string]string
	// LoadBalanceType is the load balance type.
	LoadBalanceType string

	// EnvTransfer is the environment of upstream server.
	EnvTransfer          string
	Discovery            discovery.Discovery
	DiscoveryOptions     []discovery.Option
	ServiceRouter        servicerouter.ServiceRouter
	ServiceRouterOptions []servicerouter.Option
	LoadBalancer         loadbalance.LoadBalancer
	LoadBalanceOptions   []loadbalance.Option
	CircuitBreaker       circuitbreaker.CircuitBreaker
	DisableServiceRouter bool
}

// Option modifies the Options.
type Option func(*Options)

// WithContext returns an Option which sets the request context.
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Ctx = ctx
		o.DiscoveryOptions = append(o.DiscoveryOptions, discovery.WithContext(ctx))
		o.LoadBalanceOptions = append(o.LoadBalanceOptions, loadbalance.WithContext(ctx))
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithContext(ctx))
	}
}

// WithNamespace returns an Option which sets the namespace.
func WithNamespace(namespace string) Option {
	return func(opts *Options) {
		opts.Namespace = namespace
		opts.DiscoveryOptions = append(opts.DiscoveryOptions, discovery.WithNamespace(namespace))
		opts.LoadBalanceOptions = append(opts.LoadBalanceOptions, loadbalance.WithNamespace(namespace))
		opts.ServiceRouterOptions = append(opts.ServiceRouterOptions, servicerouter.WithNamespace(namespace))
	}
}

// WithSourceSetName returns an Option which sets the set name.
func WithSourceSetName(sourceSetName string) Option {
	return func(opts *Options) {
		opts.SourceSetName = sourceSetName
		opts.ServiceRouterOptions = append(opts.ServiceRouterOptions, servicerouter.WithSourceSetName(sourceSetName))
	}
}

// WithKey returns an Option which sets the hash key of stateful routing.
func WithKey(k string) Option {
	return func(o *Options) {
		o.Key = k
		o.LoadBalanceOptions = append(o.LoadBalanceOptions, loadbalance.WithKey(k))
	}
}

// WithReplicas returns an Option which sets node replicas of stateful routing.
func WithReplicas(r int) Option {
	return func(o *Options) {
		o.Replicas = r
		o.LoadBalanceOptions = append(o.LoadBalanceOptions, loadbalance.WithReplicas(r))
	}
}

// WithDisableServiceRouter returns an Option which disables the service router.
func WithDisableServiceRouter() Option {
	return func(o *Options) {
		o.DisableServiceRouter = true
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithDisableServiceRouter())
	}
}

// WithDiscovery returns an Option which sets the discovery.
func WithDiscovery(d discovery.Discovery) Option {
	return func(o *Options) {
		o.Discovery = d
	}
}

// WithServiceRouter returns an Option which sets the service router.
func WithServiceRouter(r servicerouter.ServiceRouter) Option {
	return func(o *Options) {
		o.ServiceRouter = r
	}
}

// WithLoadBalancer returns an Option which sets load balancer.
func WithLoadBalancer(b loadbalance.LoadBalancer) Option {
	return func(o *Options) {
		o.LoadBalancer = b
	}
}

// WithLoadBalanceType returns an Option which sets load balance type.
func WithLoadBalanceType(name string) Option {
	return func(o *Options) {
		o.LoadBalanceType = name
		o.LoadBalanceOptions = append(
			o.LoadBalanceOptions,
			loadbalance.WithLoadBalanceType(name),
		)
	}
}

// WithCircuitBreaker returns an Option which sets circuit breaker.
func WithCircuitBreaker(cb circuitbreaker.CircuitBreaker) Option {
	return func(o *Options) {
		o.CircuitBreaker = cb
	}
}

// WithEnvKey returns an Option which sets the environment key.
func WithEnvKey(key string) Option {
	return func(o *Options) {
		o.EnvKey = key
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithEnvKey(key))
	}
}

// WithSourceNamespace returns an Option which sets caller namespace.
func WithSourceNamespace(namespace string) Option {
	return func(o *Options) {
		o.SourceNamespace = namespace
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithSourceNamespace(namespace))
	}
}

// WithSourceServiceName returns an Option which sets caller service name.
func WithSourceServiceName(serviceName string) Option {
	return func(o *Options) {
		o.SourceServiceName = serviceName
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithSourceServiceName(serviceName))
	}
}

// WithDestinationEnvName returns an Option which sets callee environment name.
func WithDestinationEnvName(envName string) Option {
	return func(o *Options) {
		o.DestinationEnvName = envName
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithDestinationEnvName(envName))
	}
}

// WithSourceEnvName returns an Option which sets caller environment name.
func WithSourceEnvName(envName string) Option {
	return func(o *Options) {
		o.SourceEnvName = envName
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithSourceEnvName(envName))
	}
}

// WithEnvTransfer returns an Option which sets the transparent environment information.
func WithEnvTransfer(envTransfer string) Option {
	return func(o *Options) {
		o.EnvTransfer = envTransfer
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithEnvTransfer(envTransfer))
	}
}

// WithDestinationSetName returns an Option which sets callee set name.
func WithDestinationSetName(destinationSetName string) Option {
	return func(o *Options) {
		o.DestinationSetName = destinationSetName
		o.ServiceRouterOptions = append(o.ServiceRouterOptions,
			servicerouter.WithDestinationSetName(destinationSetName))
	}
}

// WithSourceMetadata returns an Option which adds a caller metadata k-v.
// Do not use this function to set env/set, they have their own Option-s.
func WithSourceMetadata(key string, val string) Option {
	return func(o *Options) {
		if o.SourceMetadata == nil {
			o.SourceMetadata = make(map[string]string)
		}
		o.SourceMetadata[key] = val
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithSourceMetadata(key, val))
	}
}

// WithDestinationMetadata returns an Option which adds a callee metadata k-v.
// Do not use this function to set env/set, they have their own Option-s.
func WithDestinationMetadata(key string, val string) Option {
	return func(o *Options) {
		if o.DestinationMetadata == nil {
			o.DestinationMetadata = make(map[string]string)
		}
		o.DestinationMetadata[key] = val
		o.ServiceRouterOptions = append(o.ServiceRouterOptions, servicerouter.WithDestinationMetadata(key, val))
	}
}
