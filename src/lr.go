package lr

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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
	Parent: nil,
}

type LR struct {
	Tokens, fields []Token
}

// INTERFACES
type LRI interface {
	Parse(string) error
	Run(...interface{}) ([]interface{}, error)
	AST() string
	AppliesTo(...interface{}) (bool, error)

	tokenize(_ string) error
	run([]Token, ...interface{}) ([]interface{}, error)
	ast([]Token, int) string
}

var _ LRI = (*LR)(nil)

// IMPLEMENTATION
func NewParser() *LR {
	lr := &LR{
		Tokens: make([]Token, 0),
		fields: make([]Token, 0), //premature optimization
	}
	return lr
}

func resolvePath(s interface{}, path []string, forEval bool) ([]interface{}, bool) {
	if (s == nil && len(path) > 0) || len(path) == 0 {
		return nil, false
	}
	v := reflect.Indirect(reflect.ValueOf(s))
	if v.Kind() == reflect.Slice {
		if forEval {
			var r []interface{} = make([]interface{}, 0)
			for j := 0; j < v.Len(); j++ {
				ii := v.Index(j).Interface()
				res, bb := resolvePath(ii, path, forEval)
				if !bb {
					return nil, bb
				}
				r = append(r, res...)
			}
			return r, true
		} else {
			v = reflect.Indirect(v.Index(0))
		}
	}
	lci := make(map[string]int)
	for i := 0; i < v.NumField(); i++ {
		lci[strings.ToLower(v.Type().Field(i).Name)] = i
	}
	if _, ok := lci[path[0]]; ok {
		if len(path) > 1 {
			return resolvePath(v.Field(lci[strings.ToLower(path[0])]).Interface(), path[1:], forEval)
		}
		return []interface{}{v.Field(lci[strings.ToLower(path[0])]).Interface()}, true
	}
	return nil, false
}

func (lr *LR) AppliesTo(s ...interface{}) (bool, error) {
	checked := make(map[string]bool)
	for _, o := range s {
		t := reflect.TypeOf(o)
		tt := fmt.Sprint(t)
		if _, ok := checked[tt]; ok {
			continue
		}
		for _, f := range lr.fields {
			path := strings.Split(f.Value.(string), ".")
			_, b := resolvePath(o, path, false)
			if !b {
				return false, nil
			}
		}
	}
	return true, nil
}

func (lr *LR) Run(s ...interface{}) ([]interface{}, error) {
	var results []interface{} = make([]interface{}, 0)
	//for _, o := range s {
	res, err := lr.run(lr.Tokens, s...)
	if err != nil {
		return nil, err
	}
	for _, r := range res {
		results = append(results, r.(Token).Value)
	}
	//}
	return results, nil
}

func (lr *LR) run(tokens []Token, s ...interface{}) ([]interface{}, error) {
	var rval []interface{} = make([]interface{}, 0)
	for _, x := range tokens {
		switch x.Type {
		case Scope:
			res, err := lr.run(x.Value.([]Token), s...)
			if err != nil {
				return nil, err
			}
			rval = append(rval, res...)
		case FuncScope:
			args, err := lr.run(x.Value.([]Token)[1:], s...)
			if err != nil {
				return nil, err
			}
			argTokens := make([]Token, 0)
			for _, t := range args {
				argTokens = append(argTokens, t.(Token))
			}
			result, err := DefaultEnv.Values[strings.ToUpper(x.Value.([]Token)[0].Value.(string))].(func(_ []Token) (Token, error))(argTokens)
			if err != nil {
				return nil, err
			}
			rval = append(rval, result)
		case Static:
			rval = append(rval, x)
		case Field:
			for _, t := range s {
				vs, b := resolvePath(t, strings.Split(strings.ToLower(x.Value.(string)), "."), true)
				if !b {
					return nil, errors.New("Field is unresolveable")
				}
				for _, v := range vs {
					rval = append(rval, *(&Token{
						Value:    v,
						Type:     Static,
						Position: x.Position,
					}))
					//}
				}
			}
		default:
			return nil, errors.New(fmt.Sprintf("unhandled type in runner:%v\n", x.Type))
		}
	}
	return rval, nil
}

func (lr *LR) Parse(s string) error {
	return lr.tokenize(s)
}

func (lr *LR) AST() string {
	return lr.ast(lr.Tokens, 0)
}

func (lr *LR) ast(tokens []Token, depth int) string {
	bd := strings.Repeat("  ", depth)
	var ast string = ""
	for _, x := range tokens {
		switch x.Type {
		case Scope:
			ast += lr.ast(x.Value.([]Token), depth+1)
		case FuncScope:
			ast += fmt.Sprintf("%s{\n%s  \"type\": \"func\",\n%s  \"name\": \"%s\",\n%s  \"args\": [\n", bd, bd, bd, x.Value.([]Token)[0].Value.(string), bd)
			ast += lr.ast(x.Value.([]Token)[1:], depth+2)
			ast += fmt.Sprintf("\n%s  ]\n%s},\n", bd, bd)
		case Static:
			ast += fmt.Sprintf("%s{ \"type\": \"literal\", \"value\": \"%s\" },\n", bd, x.Value)
		case Field:
			ast += fmt.Sprintf("%s{ \"type\": \"field\", \"name\": \"%s\" },\n", bd, x.Value.(string))
		default:
			fmt.Printf("error in generator, unhandled type:%v\n", x.Type)
		}
	}
	trimmer := regexp.MustCompile(`,[\r\n]*$`)
	ast = trimmer.ReplaceAllString(ast, "")
	return ast
}

var opprec map[string]int = map[string]int{
	"/": 10,
	"*": 10,
	"-": 5,
	"+": 5,
	"=": 0,
}

func (lr *LR) tokenize(s string) error {
	idx := (int)(0)
	ws := regexp.MustCompile(`^[, \t\r\n]+`)
	stack := [][]Token{[]Token{}}
	for idx < len(s) {
		if m := ws.FindString(s[idx:]); len(m) > 0 {
			idx += len(m)
			continue
		}
		stacklen := len(stack) - 1
		if t, m, err := parseField(idx, s); err == nil && m > 0 {
			stack[stacklen] = append(stack[stacklen], t)
			lr.fields = append(lr.fields, t)
			idx += m
		} else if err != nil {
			return err
		} else if t, m, err := parseOperator(idx, s); err == nil && m > 0 {
			stack[stacklen] = append(stack[stacklen], t)
			idx += m
		} else if err != nil {
			return err
		} else if t, m := parseNumber(idx, s); m > 0 {
			stack[stacklen] = append(stack[stacklen], t)
			idx += m
		} else if t, m, invalid := parseStr(idx, s); !invalid && m > 0 {
			stack[stacklen] = append(stack[stacklen], t)
			idx += m
		} else if invalid {
			return errorWithLineAndPos(idx, "Unterminated string")
		} else if s[idx] == '(' {
			stack = append(stack, []Token{})
			idx++
		} else if s[idx] == ')' {
			// reduce function
			if stacklen >= 1 && len(stack[stacklen-1]) > 0 && stack[stacklen-1][len(stack[stacklen-1])-1].Type == Function {
				f := *(&Token{
					Type:     FuncScope,
					Value:    []Token{stack[stacklen-1][len(stack[stacklen-1])-1]},
					Position: stack[stacklen-1][len(stack[stacklen-1])-1].Position,
				})
				f.Value = append(f.Value.([]Token), stack[stacklen]...)
				stack[stacklen-1][len(stack[stacklen-1])-1] = f
				stack = stack[:stacklen]
			} else {
				// arbitrary scope
				v := stack[stacklen]
				stack = stack[:stacklen]
				stacklen = len(stack) - 1
				stack[stacklen] = append(stack[stacklen], *(&Token{
					Type:     Scope,
					Value:    v,
					Position: idx,
				}))
			}
			stacklen = len(stack) - 1
			idx++
		} else if t, m, invalid := parseFunction(idx, s); !invalid {
			stack[stacklen] = append(stack[stacklen], t)
			idx += m
		} else if invalid {
			return errorWithLineAndPos(idx, fmt.Sprintf("Unknown function or token: '%s'", s[idx:idx+m]))
		}

		// reducers
		// a <operator> b, with precedence
		reduce := stack[stacklen]
		reducelen := len(reduce) - 1
		if reducelen >= 2 && reduce[reducelen-1].Type == Operator {
			a := reduce[reducelen-2]
			o := reduce[reducelen-1]
			b := reduce[reducelen]
			h := false
			if a.Type == Scope || a.Type == FuncScope {
				op1, ok1 := a.Value.([]Token)[0].Value.(string)
				if ok1 {
					if prec, ok := opprec[op1]; ok {
						if prec2, ok := opprec[o.Value.(string)]; ok {
							if prec2 > prec {
								a.Value = append(a.Value.([]Token)[:len(a.Value.([]Token))-1], *(&Token{
									Type:     FuncScope,
									Value:    []Token{o, a.Value.([]Token)[len(a.Value.([]Token))-1], b},
									Position: o.Position,
								}))
								reduce[reducelen-2] = a
								reduce = reduce[:reducelen-1]
								h = true
							}
						}
					}
				}
			}
			if !h {
				reduce = append(reduce[0:reducelen-2], *(&Token{
					Type:     FuncScope,
					Value:    []Token{o, a, b},
					Position: o.Position,
				}))
			}
			stack[stacklen] = reduce
		}
	}
	if len(stack) != 1 {
		return errorWithLineAndPos(idx, "Unknown error")
	}
	if len(stack[0]) != 1 {
		return errorWithLineAndPos(stack[0][0].Position, "Unhandled reduce situation")
	}
	lr.Tokens = stack[0]
	return nil
}

func parseStr(idx int, s string) (Token, int, bool) {
	if s[idx] != '"' && s[idx] != '\'' {
		return *(&Token{}), 0, false
	}
	oidx := idx + 1
	for oidx < len(s) {
		if s[oidx] == s[idx] && s[oidx-1] != '\\' {
			break
		}
		oidx++
	}
	if oidx >= len(s) {
		return *(&Token{}), idx, true
	}
	return *(&Token{
		Type:     Static,
		Value:    s[idx+1 : oidx],
		Position: idx,
	}), oidx - idx + 1, false
}

func parseNumber(idx int, s string) (Token, int) {
	if s[idx] < '0' || s[idx] > '9' {
		return *(&Token{}), 0
	}
	oidx := idx
	foundDot := false
	for oidx < len(s) {
		if (s[oidx] < '0' || s[oidx] > '9') && (!foundDot && s[oidx] != '.') {
			break
		}
		if s[oidx] == '.' {
			foundDot = true
		}
		oidx++
	}
	v, e := strconv.ParseFloat(s[idx:oidx], 64)
	if e != nil {
		return *(&Token{}), 0
	}
	return *(&Token{
		Type:     Static,
		Value:    v,
		Position: idx,
	}), oidx - idx
}

func parseField(idx int, s string) (Token, int, error) {
	if s[idx] != '[' {
		return *(&Token{}), 0, nil
	}
	oidx := idx + 1
	for oidx < len(s) {
		if s[oidx] == ']' {
			return *(&Token{
				Type:     Field,
				Value:    (string)(s[idx+1 : oidx]),
				Position: idx,
			}), oidx - idx + 1, nil
		}
		oidx++
	}
	return *(&Token{}), idx, errorWithLineAndPos(idx, "Unterminated field")
}

func parseFunction(idx int, s string) (Token, int, bool) {
	oidx := idx
	validFuncName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z_0-9-]+`)
	if m := validFuncName.FindString(s[idx:]); len(m) > 0 {
		if _, ok := DefaultEnv.Values[strings.ToUpper(m)]; ok {
			return *(&Token{
				Type:     Function,
				Value:    m,
				Position: idx,
			}), oidx - idx + len(m), false
		}
		return *(&Token{}), oidx - idx + len(m), true
	}
	return *(&Token{}), oidx - idx + 1, true
}

func parseOperator(idx int, s string) (Token, int, error) {
	st := (string)(s[idx])
	if st == "+" || st == "-" || st == "*" || st == "/" || st == "=" || st == ">" {
		return *(&Token{
			Type:     Operator,
			Value:    st,
			Position: idx,
		}), 1, nil
	}
	return *(&Token{}), 0, nil
}

func errorWithLineAndPos(idx int, s string) error {
	lines := strings.Split(s, "\n")
	count := (int)(0)
	line := (int)(0)
	for count+len(lines[line]) < idx {
		count += len(lines[line])
		line++
	}
	return errors.New(s + fmt.Sprintf(" @ line %d, character %d", line, idx-count))
}
