package store

import (
	"strings"
)

type Name struct {
	n        string
	patterns []string
}

func NewName(name string) *Name {
	var n Name
	n.n = name
	//expand once
	n.patterns = n.expand()
	return &n
}

func (n *Name) Match(channel string) bool {
	for i := range n.patterns {
		if n.patterns[i] == channel {
			return true
		}
	}
	return false
}

func (n *Name) expand() []string {
	segments := strings.Split(n.n, "/")
	num_segments := len(segments)
	patterns := make([]string, num_segments+1)
	patterns[0] = "/**"
	for i := 1; i < len(segments); i = i + 2 {
		patterns[i] = strings.Join(segments[:i+1], "/") + "/**"
	}
	patterns[len(patterns)-2] = strings.Join(segments[:num_segments-1], "/") + "/*"
	patterns[len(patterns)-1] = n.n
	return patterns
}
