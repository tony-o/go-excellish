package fieldcalculator

import (
	"errors"
	"fmt"
	"math"
)

// TYPES
type TokenType int8

const (
	Field TokenType = iota
	Static
	Function
	Operator
	Scope
	FuncScope
)

type Token struct {
	Type     TokenType
	Value    interface{}
	Position int
}

type Env struct {
	Values map[string]interface{}
	Parent *Env
}

var operatorPrecedence map[string]int = map[string]int{
	"/": 10,
	"*": 10,
	"-": 5,
	"+": 5,
	"=": 0,
}

var DefaultEnv *Env = &Env{
	Values: map[string]interface{}{
		"SUMIF": func(ts []Token) (t Token, e error) {
			defer func() {
				if recover() != nil {
					e = errors.New("Field for SUMIF is not a number or received a bad filter")
				}
			}()
			var f float64 = 0
			filter := ts[len(ts)-1].Value.([]Token)
			for idx, t := range ts[:len(ts)-1] {
				if !filter[idx].Value.(bool) {
					continue
				}
				f += t.Value.(float64)
			}
			return *(&Token{
				Type:  Static,
				Value: f,
			}), nil
		},
		"SUM": func(ts []Token) (t Token, e error) {
			defer func() {
				if recover() != nil {
					e = errors.New("Field for SUM is not a number")
				}
			}()
			var f float64 = 0
			for _, t := range ts {
				f += t.Value.(float64)
			}
			return *(&Token{
				Type:  Static,
				Value: f,
			}), nil
		},
		"CONCAT": func(t []Token) (interface{}, error) {
			return nil, nil
		},
		"=": func(ts []Token) (Token, error) {
			l := len(ts) - 1
			var rts []Token = make([]Token, 0)
			last := ts[l].Value
			var floatVal float64
			floatCmp := false
			switch last.(type) {
			case float64:
				floatVal = last.(float64)
				floatCmp = true
			}
			for i := 0; i < l; i++ {
				cVal := ts[i].Value
				if floatCmp {
					switch cVal.(type) {
					case float64:
						if math.Abs(cVal.(float64)-floatVal) < .00001 {
							rts = append(rts, *(&Token{
								Value: true,
								Type:  Static,
							}))
						} else {
							rts = append(rts, *(&Token{
								Value: false,
								Type:  Static,
							}))
						}
						continue
					}
				}
				if fmt.Sprint(cVal) == fmt.Sprint(last) {
					rts = append(rts, *(&Token{
						Value: true,
						Type:  Static,
					}))
				} else {
					rts = append(rts, *(&Token{
						Value: false,
						Type:  Static,
					}))
				}
			}
			if l == 1 {
				return rts[0], nil
			}
			return *(&Token{
				Value: rts,
				Type:  Scope,
			}), nil
		},
		">": func(ts []Token) (Token, error) {
			l := len(ts) - 1
			var rts []Token = make([]Token, 0)
			b := ts[l].Value
			strcmp := true
			switch b.(type) {
			case float64:
				strcmp = false
			}
			for i := 0; i < l; i++ {
				a := ts[i].Value
				if !strcmp {
					switch a.(type) {
					case float64:
						if a.(float64) > b.(float64) {
							rts = append(rts, *(&Token{
								Value: true,
								Type:  Static,
							}))
						} else {
							rts = append(rts, *(&Token{
								Value: false,
								Type:  Static,
							}))
						}
						continue
					}
				}
				if fmt.Sprint(b) > fmt.Sprint(a) {
					rts = append(rts, *(&Token{
						Value: true,
						Type:  Static,
					}))
				} else {
					rts = append(rts, *(&Token{
						Value: false,
						Type:  Static,
					}))
				}
			}
			return *(&Token{
				Value: rts,
				Type:  Scope,
			}), nil
		},
		"*": func(ts []Token) (t Token, e error) {
			defer func() {
				if recover() != nil {
					e = errors.New("Field for * is not a number")
				}
			}()
			f := ts[0].Value.(float64)
			for _, t := range ts[1:] {
				f *= t.Value.(float64)
			}
			return *(&Token{
				Value: f,
				Type:  Static,
			}), nil
		},
		"/": func(ts []Token) (t Token, e error) {
			defer func() {
				if recover() != nil {
					e = errors.New("Field for / is not a number")
				}
			}()
			f := ts[0].Value.(float64)
			for _, t := range ts[1:] {
				f /= t.Value.(float64)
			}
			return *(&Token{
				Value: f,
				Type:  Static,
			}), nil
		},
		"+": func(ts []Token) (Token, error) {
			// Degradation float64 -> string
			stillf := true
			hadf := false
			var f float64
			var s string = ""
			for _, t := range ts {
				switch v := t.Value.(type) {
				default:
					if stillf {
						stillf = false
						if hadf {
							s += fmt.Sprintf("%0.000f", f)
						}
					}
					s += fmt.Sprint(v)
				case float64:
					if stillf {
						if !hadf {
							f = v
							hadf = true
						} else {
							f += v
						}
					} else {
						s += fmt.Sprintf("%0.000f", v)
					}
				}
			}
			var rv interface{}
			if stillf {
				rv = f
			} else {
				rv = s
			}
			return *(&Token{
				Value: rv,
				Type:  Static,
			}), nil
		},
	},
}

type Evaluator struct {
	Tokens, fields []Token
}

// INTERFACES
type Evaluatorizer interface {
	Parse(string) error
	Run(...interface{}) ([]interface{}, error)
	AST() string
	AppliesTo(...interface{}) (bool, error)

	tokenize(_ string) error
	run([]Token, ...interface{}) ([]interface{}, error)
	ast([]Token, int) string
}

var _ Evaluatorizer = (*Evaluator)(nil)
