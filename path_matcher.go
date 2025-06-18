package yay

import (
	"fmt"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"go.yaml.in/yaml/v3"
	"sync"
)

// PathMatcher collects information internally to wrap yamlpath.Path for optimized key/value matching during iteration.
// A user-facing PathMatcher.Match can be invoked on a node's children to determine if they also match the condition.
type PathMatcher struct {
	rawPath string
	path    *yamlpath.Path
	root    *yaml.Node
	matches map[*yaml.Node]struct{}
	mu      sync.Mutex
}

// Match determines if a node matches the yamlpath.Path condition provided by the user
func (p *PathMatcher) Match(node *yaml.Node) (bool, error) {
	err := p.ensureMatchLookup()
	if err != nil {
		return false, fmt.Errorf("path matcher lookup failed: %w", err)
	}
	if len(p.matches) == 0 {
		return false, nil
	}

	_, ok := p.matches[node]
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
		if p.matches == nil { // double-check
			p.matches = make(map[*yaml.Node]struct{})
			nodes, err := p.path.Find(p.root)
			if err != nil {
				return err
			}
			for _, n := range nodes {
				p.matches[n] = struct{}{}
			}
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
