package velocity

import (
	"net/http"
	"regexp"
	"strings"
)

type (
	nType uint8
	tree  = node
	node  struct {
		nType    nType
		prefix   string
		children map[byte]*node
		special  [catchAll + 1]*node
		endpoint *endpoint
	}
	endpoint struct {
		fn       http.HandlerFunc
		fullPath string
		pKeys    []string
	}
)

const (
	static nType = iota
	param
	catchAll
)

var paramRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func newNode(nType nType, prefix string) *node {
	return &node{
		nType:    nType,
		prefix:   prefix,
		children: make(map[byte]*node),
		special:  [catchAll + 1]*node{},
		endpoint: nil,
	}
}

func newTree() *tree {
	return &tree{
		nType:    static,
		prefix:   "",
		children: make(map[byte]*node),
		special:  [catchAll + 1]*node{},
		endpoint: nil,
	}
}

func newEndpoint(path string, fn *http.HandlerFunc, pKeys []string) *endpoint {
	return &endpoint{
		fn:       *fn,
		fullPath: path,
		pKeys:    pKeys,
	}
}

func (n *node) addChild(label byte, node *node) {
	n.children[label] = node
}

func (n *node) addSpecial(nType nType, node *node) {
	n.special[nType] = node
}

func (n *node) clearChildren() {
	n.children = make(map[byte]*node)
	n.special = [catchAll + 1]*node{}
}

func (n *node) copyFrom(node *node) {
	n.children = node.children
	n.special = node.special
	n.endpoint = node.endpoint
}

func (n *node) setEndpoint(e *endpoint) {
	n.endpoint = e
}

func (t *tree) insert(p string, fn http.HandlerFunc) {
	p = cleanPath(p)
	if !isValidPath(p) {
		return
	}
	cur := t
	pKeys := []string{}
	for _, seg := range splitPath(p) {
		switch getSegmentType(seg) {
		case static:
			search := seg
			for len(search) > 0 {
				next := cur.children[search[0]]
				if next == nil {
					new := newNode(static, search)
					cur.addChild(search[0], new)
					cur = new
					search = ""
					continue
				}
				lcp := longestPrefix(search, next.prefix)

				if len(lcp) < len(next.prefix) {
					rem := next.prefix[len(lcp):]
					next.prefix = lcp

					new := newNode(static, rem)
					new.copyFrom(next)
					next.clearChildren()
					next.endpoint = nil
					next.addChild(rem[0], new)
					if len(search) > len(lcp) {
						newSearch := newNode(static, search[len(lcp):])
						next.addChild(search[len(lcp):][0], newSearch)
						cur = newSearch
					} else {
						cur = next
					}
					search = ""
					continue
				}
				cur = next
				search = search[len(lcp):]
			}
		case param:
			pKeys = append(pKeys, seg[1:])
			n := cur.special[param]
			if n == nil {
				new := newNode(param, "")
				cur.addSpecial(param, new)
				cur = new
				continue
			}
			cur = n
		case catchAll:
			pKeys = append(pKeys, "*")
			n := cur.special[catchAll]
			if n == nil {
				new := newNode(catchAll, "")
				cur.addSpecial(catchAll, new)
				cur = new
				continue
			}
			cur = n
		}

	}
	e := newEndpoint(p, &fn, pKeys)
	cur.setEndpoint(e)
}

func (t *tree) find(p string) (*endpoint, map[string]string) {
	params := []string{}
	cur := t
	var prevLabel *byte
	start := 0
	for len(p) > 0 {

		if p[0] == '/' {
			p = p[1:]
			continue
		}

		label := p[0]
		if prevLabel != nil {
			label = *prevLabel
		}

		if static := cur.children[label]; static != nil {
			lcp := longestPrefix(p, static.prefix[start:])
			if len(lcp) == 0 {
				break
			}
			if len(lcp)+start == len(static.prefix) {
				cur = static
				p = p[len(lcp):]
				prevLabel = nil
				start = 0
			} else {
				prevLabel = &label
				start += len(lcp)
				p = p[len(lcp):]
			}
			continue
		}

		if param := cur.special[param]; param != nil {
			j := strings.IndexByte(p, '/')
			if j == -1 {
				params = append(params, p)
				cur = param
				p = ""
			} else {
				params = append(params, p[:j])
				cur = param
				p = p[j+1:]
			}
			continue
		}

		if catchAll := cur.special[catchAll]; catchAll != nil {
			params = append(params, p)
			cur = catchAll
			p = ""
			continue
		}
		return nil, map[string]string{}
	}

	if cur.endpoint == nil {
		return nil, map[string]string{}
	}

	pMap := map[string]string{}
	for i, k := range cur.endpoint.pKeys {
		pMap[k] = params[i]
	}

	return cur.endpoint, pMap
}

func splitPath(p string) []string {
	p = strings.TrimPrefix(p, "/")
	segments := []string{}
	cur := ""
	for _, seg := range strings.Split(p, "/") {
		switch getSegmentType(seg) {
		case static:
			cur += seg
		default:
			if cur != "" {
				segments = append(segments, cur)
			}
			segments = append(segments, seg)
			cur = ""
		}
	}
	if cur != "" {
		segments = append(segments, cur)
	}
	return segments
}

func cleanPath(p string) string {
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, "/")
	segments := strings.Split(p, "/")
	final := []string{}
	for _, seg := range segments {
		if seg != "" {
			final = append(final, seg)
		}
	}
	return "/" + strings.Join(final, "/")
}

func getSegmentType(s string) nType {
	switch {
	case s[0] == ':':
		return param
	case s[0] == '*':
		return catchAll
	default:
		return static
	}
}

func longestPrefix(s1, s2 string) string {
	min := len(s1)
	if len(s2) < min {
		min = len(s2)
	}
	for i := 0; i < min; i++ {
		if s1[i] != s2[i] {
			return s1[:i]
		}
	}
	return s1[:min]
}

func isValidPath(p string) bool {
	var prevTyp *nType
	segments := splitPath(p)
	keys := map[string]struct{}{}
	for i, seg := range segments {
		if seg == "" {
			continue
		}
		typ := getSegmentType(seg)
		// Cannot have two variadic segments together
		if prevTyp != nil && *prevTyp != static && typ != static {
			return false
		}
		// Catch-all must be last
		if typ == catchAll && i != len(segments)-1 {
			return false
		}
		// Cannot have repeat param keys
		if typ == param {
			_, ok := keys[seg]
			if ok {
				return false
			}
			keys[seg] = struct{}{}
		}
		// Is invalid param name
		if typ == param && !paramRegex.MatchString(seg[1:]) {
			return false
		}
		// Catch all may only contain "*"
		if typ == catchAll && seg != "*" {
			return false
		}
		prevTyp = &typ
	}
	return true
}

func (t *tree) captureRoutes(m string) []string {
	return recurseCapture(m, t, []string{})
}

func recurseCapture(m string, n *node, r []string) []string {
	for _, c := range n.special {
		if c == nil {
			continue
		}
		if c.endpoint != nil {
			r = append(r, m+" "+c.endpoint.fullPath)
		}
		r = recurseCapture(m, c, r)
	}
	for _, c := range n.children {
		if c.endpoint != nil {
			r = append(r, m+" "+c.endpoint.fullPath)
		}
		r = recurseCapture(m, c, r)
	}
	return r
}
