package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConditionParseTrue(t *testing.T) {

	exp, err := GetExpression("true")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, exp.Examine(nil)); !ok {
		t.Error()
	}
}

func TestConditionParseFalse(t *testing.T) {

	exp, err := GetExpression("false")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, exp.Examine(nil)); !ok {
		t.Error()
	}
}

func TestConditionParseAnd(t *testing.T) {

	exp, err := GetExpression("and(true, false)")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, exp.Examine(nil)); !ok {
		t.Error()
	}

	exp, err = GetExpression("and(true, true)")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, exp.Examine(nil)); !ok {
		t.Error()
	}
}

func TestConditionParseOr(t *testing.T) {

	exp, err := GetExpression("or(true, false)")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, exp.Examine(nil)); !ok {
		t.Error()
	}

	exp, err = GetExpression("or(false, false)")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.False(t, exp.Examine(nil)); !ok {
		t.Error()
	}
}

func TestConditionParseComplex(t *testing.T) {

	exp, err := GetExpression("and(true, or(not(false), false))")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, exp.Examine(nil)); !ok {
		t.Error()
	}
}

func TestConditionParseComplexUserInput(t *testing.T) {

	exp, err := GetExpression("and(true, or(not(user-input), false))")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, exp.Examine(nil)); !ok {
		t.Error()
	}

	input := map[string]interface{}{
		"user-input": "1",
	}

	if ok := assert.False(t, exp.Examine(input)); !ok {
		t.Error()
	}
}

func TestConditionParseComplexUserInputSpaces(t *testing.T) {

	exp, err := GetExpression(" and (true, or( not(user-input) , false ) )")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, exp.Examine(nil)); !ok {
		t.Error()
	}

	input := map[string]interface{}{
		"user-input": "1",
	}

	if ok := assert.False(t, exp.Examine(input)); !ok {
		t.Error()
	}
}

func TestConditionParseComplexExmpty(t *testing.T) {

	exp, err := GetExpression("")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.True(t, exp.Examine(nil)); !ok {
		t.Error()
	}
}
