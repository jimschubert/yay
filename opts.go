package yay

type opts struct {
	initialized       bool
	skipDocumentCheck bool
}

// FnOptions is a function chain of options to apply conditionally to a Visitor
type FnOptions func(o *opts)

// WithSkipDocumentCheck allows the user to configure a Visitor to skip the requirement that a top-level
// node must be a yaml.DocumentNode. This allows creating and invoking visitors against manually constructed nodes.
func (fn FnOptions) WithSkipDocumentCheck(val bool) FnOptions {
	return func(o *opts) {
		fn(o)
		o.skipDocumentCheck = val
	}
}

// NewOptions creates a new options functional builder with discoverable functions that don't pollute the yay package
//
//goland:noinspection GoExportedFuncWithUnexportedType
func NewOptions() FnOptions {
	return func(o *opts) {
		if !o.initialized {
			o.skipDocumentCheck = false
			o.initialized = true
		}
	}
}
