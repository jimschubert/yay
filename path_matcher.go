package yay

import (
	"sync"
	"unsafe"

	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

// PathMatcher collects information internally to wrap yamlpath.Path for optimized key/value matching during iteration.
// A user-facing PathMatcher.Match can be invoked on a node's children to determine if they also match the condition.
type PathMatcher struct {
	rawPath string
	path    *yamlpath.Path
	root    *yaml.Node
	matches map[uintptr]struct{}
	mu      sync.Mutex
}

// Match determines if a node matches the yamlpath.Path condition provided by the user
func (p *PathMatcher) Match(node *yaml.Node) (bool, error) {
	if err := p.ensureMatchLookup(); err != nil || len(p.matches) == 0 {
		return false, err
	}

	_, ok := p.matches[uintptr(unsafe.Pointer(node))]
	return ok, nil
}

// MustMatch is exactly like Match, but panics if the expected match fails
func (p *PathMatcher) MustMatch(node *yaml.Node) bool {
	result, err := p.Match(node)
	if err != nil {
		panic(err.Error())
	}
	return result
}

func (p *PathMatcher) ensureMatchLookup() error {
	if p.matches == nil {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.matches = make(map[uintptr]struct{})
		nodes, err := p.path.Find(p.root)
		if err != nil {
			return err
		}
		for _, n := range nodes {
			p.matches[uintptr(unsafe.Pointer(n))] = struct{}{}
		}
	}
	return nil
}

func newPathMatcher(path string) (*PathMatcher, error) {
	yp, err := yamlpath.NewPath(path)
	if err != nil {
		return nil, err
	}
	return &PathMatcher{rawPath: path, path: yp}, nil
}
