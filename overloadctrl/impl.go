package overloadctrl

import (
	"context"
	"fmt"
)

// Impl provides a YAML-configurable overload controller implementation.
type Impl struct {
	OverloadController        // exported for tests and advanced integrations.
	Builder            string // exported for server backward compatibility.
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (impl *Impl) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&impl.Builder)
}

// MarshalYAML implements yaml.Marshaler.
func (impl Impl) MarshalYAML() (interface{}, error) {
	return impl.Builder, nil
}

// Acquire implements OverloadController.
func (impl *Impl) Acquire(ctx context.Context, addr string) (Token, error) {
	if impl.OverloadController == nil {
		return NoopToken{}, nil
	}
	return impl.OverloadController.Acquire(ctx, addr)
}

// Build constructs the actual overload controller instance.
func (impl *Impl) Build(getBuilder func(string) Builder, smi *ServiceMethodInfo) error {
	if impl.Builder == "" {
		return nil
	}
	newOC := getBuilder(impl.Builder)
	if newOC == nil {
		return fmt.Errorf("overload control builder %s is not found", impl.Builder)
	}
	impl.OverloadController = newOC(smi)
	return nil
}
