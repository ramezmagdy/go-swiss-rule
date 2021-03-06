package go_swiss_rule

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/karim-albakry/go-swiss-rule/utils/errors"
	"reflect"
	"strings"
)

func nestedMapLookup(m map[string]interface{}, ks ...string) (rval interface{}, err error) {
	var ok bool

	if len(ks) == 0 { // degenerate input
		return nil, fmt.Errorf("NestedMapLookup needs at least one key")
	}
	if rval, ok = m[ks[0]]; !ok {
		return nil, fmt.Errorf("key not found; remaining keys: %v", ks)
	} else if len(ks) == 1 { // we've reached the final key
		return rval, nil
	} else if m, ok = rval.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("malformed structure at %#v", rval)
	} else { // 1+ more keys
		return nestedMapLookup(m, ks[1:]...)
	}
}

func buildStringQuery(input map[string]interface{}, rule Rule) (result string, err error) {
	for index, condition := range rule.Conditions {
		args := strings.Split(condition.Key, ".")
		value, lookupErr := nestedMapLookup(input, args...)
		if lookupErr != nil {
			result = ""
			err = lookupErr
			return
		}
		valueType := reflect.
			TypeOf(value).
			String()
		if valueType == "string" {
			result += fmt.
				Sprintf("(('%v') %s '%v') ",
					value, condition.Operator,
					condition.Value)
		} else {
			result += fmt.
				Sprintf("(%v %s %v) ",
					value,
					condition.Operator,
					condition.Value)
		}
		if index !=
			(len(rule.Conditions)-1) && condition.Joint != "" {
			result += fmt.
				Sprintf(" %v ", condition.Joint)
		}
	}
	return
}

func invokeActions(rule Rule) error {
	for _, action := range rule.Actions {
		if err := action.Fire(); err != nil {
			return err
		}
	}
	return nil
}

func EvalAndInvoke(input map[string]interface{}, rule Rule) (bool, error) {
	if len(input) < 0 {
		return false, errors.SimpleError("Insufficient input.")
	}
	constraints, err := buildStringQuery(input, rule)
	if err != nil {
		return false, err
	}
	if constraints != "" {
		result, exprError := expr.Eval(constraints, input)
		if exprError != nil {
			return false, exprError
		}
		if result == true {
			err := invokeActions(rule)
			if err != nil {
				return false, err
			}
		}
		return result == true, nil
	}
	return false, errors.SimpleError("No conditions to execute, review your conditions keys.")
}
