package expr

import (
	"sort"

	"github.com/dwlhm/nova/internal/core/lexer"
	"github.com/dwlhm/nova/internal/core/parser"
)

// Func holds a pure Nova function body for interpretation and lowering.
type Func struct {
	Name   string
	Params []string
	Body   []lexer.Token
}

// Registry maps function names to pure func definitions from parsed sources.
type Registry struct {
	funcs map[string]Func
}

func NewRegistry() *Registry {
	return &Registry{funcs: make(map[string]Func)}
}

func BuildRegistry(files []parser.File) *Registry {
	reg := NewRegistry()
	for _, file := range files {
		for _, fn := range file.Funcs {
			params := make([]string, 0, len(fn.Params))
			for _, param := range fn.Params {
				params = append(params, param.Name)
			}
			reg.funcs[fn.Name] = Func{
				Name:   fn.Name,
				Params: params,
				Body:   append([]lexer.Token(nil), fn.Body...),
			}
		}
	}
	return reg
}

func (r *Registry) Lookup(name string) (Func, bool) {
	if r == nil {
		return Func{}, false
	}
	fn, ok := r.funcs[name]
	return fn, ok
}

func (r *Registry) Names() []string {
	if r == nil || len(r.funcs) == 0 {
		return nil
	}
	names := make([]string, 0, len(r.funcs))
	for name := range r.funcs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
