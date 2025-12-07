package capacitor

import (
	"fmt"
	"time"
)

// Config defines the configuration for a multi-layer DAL.
type Config[K comparable, V any] struct {
	// Layers defines the storage layers in order (fastest to slowest).
	Layers []LayerConfig[K, V]

	// EnableMetrics enables performance metrics collection.
	EnableMetrics bool

	// EnablePromotion enables automatic promotion of values to faster layers on cache hits.
	EnablePromotion bool

	// WriteThrough controls write behavior:
	// - true: writes propagate to all layers synchronously
	// - false: writes only go to fastest layer (not recommended)
	WriteThrough bool
}

// LayerConfig defines configuration for a single layer.
type LayerConfig[K comparable, V any] struct {
	// Name identifies this layer (e.g., "L1", "L2", "persistent").
	Name string

	// Layer is the actual layer implementation.
	Layer Layer[K, V]

	// TTL is the default time-to-live for values in this layer.
	// Zero means no expiration.
	TTL time.Duration

	// ReadOnly indicates this layer should not be written to.
	// Useful for read-only database replicas.
	ReadOnly bool
}

// Builder provides a fluent API for constructing a DAL configuration.
type Builder[K comparable, V any] struct {
	config Config[K, V]
	err    error
}

// NewBuilder creates a new configuration builder.
func NewBuilder[K comparable, V any]() *Builder[K, V] {
	return &Builder[K, V]{
		config: Config[K, V]{
			Layers:          make([]LayerConfig[K, V], 0),
			EnableMetrics:   true,
			EnablePromotion: true,
			WriteThrough:    true,
		},
	}
}

// WithLayer adds a layer to the configuration.
// Layers should be added in order from fastest to slowest.
func (b *Builder[K, V]) WithLayer(name string, layer Layer[K, V], ttl time.Duration) *Builder[K, V] {
	if b.err != nil {
		return b
	}

	if name == "" {
		b.err = fmt.Errorf("layer name cannot be empty")
		return b
	}

	if layer == nil {
		b.err = fmt.Errorf("layer cannot be nil")
		return b
	}

	// Check for duplicate layer names
	for _, l := range b.config.Layers {
		if l.Name == name {
			b.err = fmt.Errorf("duplicate layer name: %s", name)
			return b
		}
	}

	b.config.Layers = append(b.config.Layers, LayerConfig[K, V]{
		Name:     name,
		Layer:    layer,
		TTL:      ttl,
		ReadOnly: false,
	})

	return b
}

// WithReadOnlyLayer adds a read-only layer to the configuration.
func (b *Builder[K, V]) WithReadOnlyLayer(name string, layer Layer[K, V]) *Builder[K, V] {
	if b.err != nil {
		return b
	}

	if name == "" {
		b.err = fmt.Errorf("layer name cannot be empty")
		return b
	}

	if layer == nil {
		b.err = fmt.Errorf("layer cannot be nil")
		return b
	}

	b.config.Layers = append(b.config.Layers, LayerConfig[K, V]{
		Name:     name,
		Layer:    layer,
		TTL:      0,
		ReadOnly: true,
	})

	return b
}

// WithMetrics enables or disables metrics collection.
func (b *Builder[K, V]) WithMetrics(enabled bool) *Builder[K, V] {
	if b.err != nil {
		return b
	}

	b.config.EnableMetrics = enabled
	return b
}

// WithPromotion enables or disables automatic value promotion.
func (b *Builder[K, V]) WithPromotion(enabled bool) *Builder[K, V] {
	if b.err != nil {
		return b
	}

	b.config.EnablePromotion = enabled
	return b
}

// WithWriteThrough configures write-through behavior.
func (b *Builder[K, V]) WithWriteThrough(enabled bool) *Builder[K, V] {
	if b.err != nil {
		return b
	}

	b.config.WriteThrough = enabled
	return b
}

// Build constructs the final configuration.
// Returns an error if the configuration is invalid.
func (b *Builder[K, V]) Build() (Config[K, V], error) {
	if b.err != nil {
		return Config[K, V]{}, b.err
	}

	if len(b.config.Layers) == 0 {
		return Config[K, V]{}, fmt.Errorf("at least one layer must be configured")
	}

	return b.config, nil
}

// Validate checks if the configuration is valid.
func (c *Config[K, V]) Validate() error {
	if len(c.Layers) == 0 {
		return fmt.Errorf("at least one layer must be configured")
	}

	names := make(map[string]bool)
	for _, layer := range c.Layers {
		if layer.Name == "" {
			return fmt.Errorf("layer name cannot be empty")
		}

		if layer.Layer == nil {
			return fmt.Errorf("layer %s: implementation cannot be nil", layer.Name)
		}

		if names[layer.Name] {
			return fmt.Errorf("duplicate layer name: %s", layer.Name)
		}

		names[layer.Name] = true
	}

	return nil
}
