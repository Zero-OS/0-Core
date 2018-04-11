package settings

import (
	"fmt"
	"regexp"
	"strings"
)

type expBuilder func(args []Expression) Expression

var (
	word = regexp.MustCompile(`^[\w-]+`)

	//EOL parser has reached end of expression
	EOL = fmt.Errorf("end of line")

	expressions = map[string]expBuilder{
		"true":  func(_ []Expression) Expression { return boolExp(true) },
		"false": func(_ []Expression) Expression { return boolExp(false) },
		"and":   func(args []Expression) Expression { return andExp{args} },
		"or":    func(args []Expression) Expression { return orExp{args} },
		"not":   func(args []Expression) Expression { return notExp{args} },
	}
)

//Expression represents a loaded expression
type Expression interface {
	Examine(input map[string]interface{}) bool
}

type boolExp bool

func (t boolExp) Examine(_ map[string]interface{}) bool {
	return bool(t)
}

type andExp struct {
	Args []Expression
}

func (t andExp) Examine(in map[string]interface{}) bool {
	for _, arg := range t.Args {
		if !arg.Examine(in) {
			return false
		}
	}
	return true
}

func (t andExp) String() string {
	var l []string
	for _, a := range t.Args {
		l = append(l, fmt.Sprint(a))
	}
	return fmt.Sprintf("AND (%s)", strings.Join(l, ", "))
}

type orExp struct {
	Args []Expression
}

func (t orExp) Examine(in map[string]interface{}) bool {
	for _, arg := range t.Args {
		if arg.Examine(in) {
			return true
		}
	}

	return false
}

func (t orExp) String() string {
	var l []string
	for _, a := range t.Args {
		l = append(l, fmt.Sprint(a))
	}
	return fmt.Sprintf("OR (%s)", strings.Join(l, ", "))
}

type notExp struct {
	Args []Expression
}

func (t notExp) Examine(in map[string]interface{}) bool {
	if len(t.Args) != 1 {
		return false
	}

	return !t.Args[0].Examine(in)
}

func (t notExp) String() string {
	var l []string
	for _, a := range t.Args {
		l = append(l, fmt.Sprint(a))
	}
	return fmt.Sprintf("NOT (%s)", strings.Join(l, ", "))
}

type userExp string

func (u userExp) Examine(in map[string]interface{}) bool {
	_, ok := in[string(u)]
	return ok
}

//forward return position of the first non space char starting at from
func forward(from int, s string) int {
	return strings.IndexFunc(s[from:], func(c rune) bool {
		return c != ' '
	}) + from
}

func getOne(at int, expression string) (Expression, int, error) {
	at = strings.IndexFunc(expression[at:], func(c rune) bool {
		return c != ' '
	}) + at

	loc := word.FindStringIndex(expression[at:])
	if loc == nil {
		return nil, at, nil
	}

	token := expression[loc[0]+at : loc[1]+at]
	next := loc[1] + at

	next = strings.IndexFunc(expression[at:], func(c rune) bool {
		return c != ' '
	}) + next

	var args []Expression

	if next < len(expression)-1 {
		if expression[next] == '(' {
			for {
				var sub Expression
				var err error
				sub, next, err = getOne(next+1, expression)
				if err != nil {
					return nil, next, err
				}
				args = append(args, sub)

				//find next non space char
				next = strings.IndexFunc(expression[next:], func(c rune) bool {
					return c != ' '
				}) + next

				if expression[next] == ')' {
					next++
					break
				} else if expression[next] != ',' {
					return nil, next, fmt.Errorf("expecting , or )")
				}
			}
		}
	}

	builder, ok := expressions[token]
	var exp Expression
	var err error
	if ok {
		exp = builder(args)
	} else {
		exp = userExp(token)
	}

	if next >= len(expression) {
		err = EOL
	}

	return exp, next, err
}

//GetExpression gets an expression object from string, empty string always evaluates
//to a `true` expression.
func GetExpression(expression string) (Expression, error) {
	if len(expression) == 0 {
		//Note, an empty expression is evaluated as true
		//So services witch does not define conditions, runs by default
		return boolExp(true), nil
	}

	exp, l, err := getOne(0, expression)
	if err == EOL && exp != nil {
		return exp, nil
	} else if l != len(expression) {
		return nil, fmt.Errorf("garbage at end of expression: '%s' pos: %d", expression, l)
	} else {
		return nil, fmt.Errorf("syntax error(%s): '%s' pos: %d", err, expression, l)
	}
}
