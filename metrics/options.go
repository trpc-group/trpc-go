package metrics

var options = &Options{}

// Options defines the report options.
type Options struct {
	// Meta is used to adapt some complex scenes. For example, a monitor may map the metric name to
	// monitor id.
	Meta map[string]interface{}
}

// GetOptions gets options.
func GetOptions() Options {
	return *options
}

// Option modifies the Options.
type Option func(opts *Options)

// WithMeta returns an Option which sets the metadata, such as a map between metric name and metric
// id.
func WithMeta(meta map[string]interface{}) Option {
	return func(opts *Options) {
		if opts != nil {
			opts.Meta = meta
		}
	}
}
