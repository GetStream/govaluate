package govaluate

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
)

const (
	logicalErrorFormat    string = "Value '%v' cannot be used with the logical operator '%v', it is not a bool"
	modifierErrorFormat   string = "Value '%v' cannot be used with the modifier '%v', it is not a number"
	comparatorErrorFormat string = "Value '%v' cannot be used with the comparator '%v', it is not a number"
	ternaryErrorFormat    string = "Value '%v' cannot be used with the ternary operator '%v', it is not a bool"
	prefixErrorFormat     string = "Value '%v' cannot be used with the prefix '%v'"
)

type evaluationOperator func(left, right interface{}, parameters Parameters) (interface{}, error)
type stageTypeCheck func(value interface{}) bool
type stageCombinedTypeCheck func(left, right interface{}) bool

type evaluationStage struct {
	symbol OperatorSymbol

	leftStage, rightStage *evaluationStage

	// the operation that will be used to evaluate this stage (such as adding [left] to [right] and return the result)
	operator evaluationOperator

	// ensures that both left and right values are appropriate for this stage. Returns an error if they aren't operable.
	leftTypeCheck, rightTypeCheck stageTypeCheck

	// if specified, will override whatever is used in "leftTypeCheck" and "rightTypeCheck".
	// primarily used for specific operators that don't care which side a given type is on, but still requires one side to be of a given type
	// (like string concat)
	typeCheck stageCombinedTypeCheck

	// regardless of which type check is used, this string format will be used as the error message for type errors
	typeErrorFormat string
}

var (
	_true  = interface{}(true)
	_false = interface{}(false)
)

func (s *evaluationStage) swapWith(other *evaluationStage) {
	temp := *other
	other.setToNonStage(*s)
	s.setToNonStage(temp)
}

func (s *evaluationStage) setToNonStage(other evaluationStage) {
	s.symbol = other.symbol
	s.operator = other.operator
	s.leftTypeCheck = other.leftTypeCheck
	s.rightTypeCheck = other.rightTypeCheck
	s.typeCheck = other.typeCheck
	s.typeErrorFormat = other.typeErrorFormat
}

func (s *evaluationStage) isShortCircuitable() bool {
	switch s.symbol {
	case AND, OR, TERNARY_TRUE, TERNARY_FALSE, COALESCE:
		return true
	}

	return false
}

func noopStageRight(left, right interface{}, parameters Parameters) (interface{}, error) {
	return right, nil
}

func addStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	// string concat if either are strings
	if isString(left) || isString(right) {
		return fmt.Sprintf("%v%v", left, right), nil
	}

	return left.(float64) + right.(float64), nil
}
func subtractStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return left.(float64) - right.(float64), nil
}
func multiplyStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return left.(float64) * right.(float64), nil
}
func divideStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return left.(float64) / right.(float64), nil
}
func exponentStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return math.Pow(left.(float64), right.(float64)), nil
}
func modulusStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return math.Mod(left.(float64), right.(float64)), nil
}
func gteStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	if isString(left) && isString(right) {
		return boolIface(left.(string) >= right.(string)), nil
	}
	return boolIface(left.(float64) >= right.(float64)), nil
}
func gtStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	if isString(left) && isString(right) {
		return boolIface(left.(string) > right.(string)), nil
	}
	return boolIface(left.(float64) > right.(float64)), nil
}
func lteStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	if isString(left) && isString(right) {
		return boolIface(left.(string) <= right.(string)), nil
	}
	return boolIface(left.(float64) <= right.(float64)), nil
}
func ltStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	if isString(left) && isString(right) {
		return boolIface(left.(string) < right.(string)), nil
	}
	return boolIface(left.(float64) < right.(float64)), nil
}
func equalStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return boolIface(reflect.DeepEqual(left, right)), nil
}
func notEqualStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return boolIface(!reflect.DeepEqual(left, right)), nil
}
func andStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return boolIface(left.(bool) && right.(bool)), nil
}
func orStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return boolIface(left.(bool) || right.(bool)), nil
}
func negateStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return -right.(float64), nil
}
func invertStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return boolIface(!right.(bool)), nil
}
func bitwiseNotStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(^int64(right.(float64))), nil
}
func ternaryIfStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	if left.(bool) {
		return right, nil
	}
	return nil, nil
}
func ternaryElseStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	if left != nil {
		return left, nil
	}
	return right, nil
}

func regexStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	var pattern *regexp.Regexp
	var err error

	switch right := right.(type) {
	case string:
		pattern, err = regexp.Compile(right)
		if err != nil {
			return nil, fmt.Errorf("Unable to compile regexp pattern '%v': %v", right, err)
		}
	case *regexp.Regexp:
		pattern = right
	}

	return pattern.Match([]byte(left.(string))), nil
}

func notRegexStage(left, right interface{}, parameters Parameters) (interface{}, error) {

	ret, err := regexStage(left, right, parameters)
	if err != nil {
		return nil, err
	}

	return !(ret.(bool)), nil
}

func bitwiseOrStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(int64(left.(float64)) | int64(right.(float64))), nil
}
func bitwiseAndStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(int64(left.(float64)) & int64(right.(float64))), nil
}
func bitwiseXORStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(int64(left.(float64)) ^ int64(right.(float64))), nil
}
func leftShiftStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(uint64(left.(float64)) << uint64(right.(float64))), nil
}
func rightShiftStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	return float64(uint64(left.(float64)) >> uint64(right.(float64))), nil
}

func makeParameterStage(parameterName string) evaluationOperator {
	return func(left, right interface{}, parameters Parameters) (interface{}, error) {
		value, err := parameters.Get(parameterName)
		if err != nil {
			return nil, err
		}

		return value, nil
	}
}

func makeLiteralStage(literal interface{}) evaluationOperator {
	return func(left, right interface{}, parameters Parameters) (interface{}, error) {
		return literal, nil
	}
}

func makeFunctionStage(function ExpressionFunction) evaluationOperator {
	return func(left, right interface{}, parameters Parameters) (interface{}, error) {
		if right == nil {
			return function()
		}

		switch right := right.(type) {
		case []interface{}:
			return function(right...)
		default:
			return function(right)
		}
	}
}

func typeConvertParam(p reflect.Value, t reflect.Type) (ret reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Argument type conversion failed: failed to convert '%s' to '%s'", p.Kind().String(), t.Kind().String())
			ret = p
		}
	}()

	return p.Convert(t), nil
}

func typeConvertParams(method reflect.Value, params []reflect.Value) ([]reflect.Value, error) {
	methodType := method.Type()
	numIn := methodType.NumIn()
	numParams := len(params)

	if numIn != numParams {
		if numIn > numParams {
			return nil, fmt.Errorf("Too few arguments to parameter call: got %d arguments, expected %d", len(params), numIn)
		}
		return nil, fmt.Errorf("Too many arguments to parameter call: got %d arguments, expected %d", len(params), numIn)
	}

	for i := 0; i < numIn; i++ {
		t := methodType.In(i)
		p := params[i]
		pt := p.Type()

		if t.Kind() != pt.Kind() {
			np, err := typeConvertParam(p, t)
			if err != nil {
				return nil, err
			}
			params[i] = np
		}
	}

	return params, nil
}

//nolint: gocognit
func makeAccessorStage(pair []string) evaluationOperator {
	reconstructed := strings.Join(pair, ".")

	return func(left interface{}, right interface{}, parameters Parameters) (ret interface{}, err error) {

		var params []reflect.Value

		value, err := parameters.Get(pair[0])
		if err != nil {
			return nil, err
		}

		// while this library generally tries to handle panic-inducing cases on its own,
		// accessors are a sticky case which have a lot of possible ways to fail.
		// therefore every call to an accessor sets up a defer that tries to recover from panics, converting them to errors.
		defer func() {
			if r := recover(); r != nil {
				errorMsg := fmt.Sprintf("Failed to access '%s': %v", reconstructed, r.(string))
				err = errors.New(errorMsg)
				ret = nil
			}
		}()

		for i := 1; i < len(pair); i++ {

			coreValue := reflect.ValueOf(value)

			var corePtrVal reflect.Value

			// if this is a pointer, resolve it.
			if coreValue.Kind() == reflect.Ptr {
				corePtrVal = coreValue
				coreValue = coreValue.Elem()
			}

			if coreValue.Kind() != reflect.Struct {
				return nil, errors.New("Unable to access '" + pair[i] + "', '" + pair[i-1] + "' is not a struct")
			}

			field := coreValue.FieldByName(pair[i])
			if field != (reflect.Value{}) {
				value = field.Interface()
				continue
			}

			method := coreValue.MethodByName(pair[i])
			if method == (reflect.Value{}) {
				if corePtrVal.IsValid() {
					method = corePtrVal.MethodByName(pair[i])
				}
				if method == (reflect.Value{}) {
					return nil, errors.New("No method or field '" + pair[i] + "' present on parameter '" + pair[i-1] + "'")
				}
			}

			switch right := right.(type) {
			case []interface{}:
				givenParams := right
				params = make([]reflect.Value, len(givenParams))
				for idx := range givenParams {
					params[idx] = reflect.ValueOf(givenParams[idx])
				}

			default:
				if right == nil {
					params = []reflect.Value{}
					break
				}
				params = []reflect.Value{reflect.ValueOf(right)}
			}

			params, err = typeConvertParams(method, params)

			if err != nil {
				return nil, errors.New("Method call failed - '" + pair[0] + "." + pair[1] + "': " + err.Error())
			}

			returned := method.Call(params)
			retLength := len(returned)

			if retLength == 0 {
				return nil, errors.New("Method call '" + pair[i-1] + "." + pair[i] + "' did not return any values.")
			}

			if retLength == 1 {

				value = returned[0].Interface()
				continue
			}

			if retLength == 2 {

				errIface := returned[1].Interface()
				err, validType := errIface.(error)

				if validType && errIface != nil {
					return returned[0].Interface(), err
				}

				value = returned[0].Interface()
				continue
			}

			return nil, errors.New("Method call '" + pair[0] + "." + pair[1] + "' did not return either one value, or a value and an error. Cannot interpret meaning.")
		}

		value = castToFloat64(value)
		return value, nil
	}
}

func separatorStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	switch left := left.(type) {
	case []interface{}:
		return append(left, right), nil
	default:
		return []interface{}{left, right}, nil
	}
}

func inStage(left, right interface{}, parameters Parameters) (interface{}, error) {
	for _, value := range right.([]interface{}) {
		if left == value {
			return true, nil
		}
	}
	return false, nil
}

//

func isString(value interface{}) bool {
	_, ok := value.(string)
	return ok
}

func isRegexOrString(value interface{}) bool {
	switch value.(type) {
	case string:
		return true
	case *regexp.Regexp:
		return true
	}
	return false
}

func isBool(value interface{}) bool {
	_, ok := value.(bool)
	return ok
}

func isFloat64(value interface{}) bool {
	_, ok := value.(float64)
	return ok
}

/*
	Addition usually means between numbers, but can also mean string concat.
	String concat needs one (or both) of the sides to be a string.
*/
func additionTypeCheck(left, right interface{}) bool {
	if isFloat64(left) && isFloat64(right) {
		return true
	}
	return isString(left) || isString(right)
}

/*
	Comparison can either be between numbers, or lexicographic between two strings,
	but never between the two.
*/
func comparatorTypeCheck(left, right interface{}) bool {
	if isFloat64(left) && isFloat64(right) {
		return true
	}
	return isString(left) && isString(right)
}

func isArray(value interface{}) bool {
	_, ok := value.([]interface{})
	return ok
}

/*
	Converting a boolean to an interface{} requires an allocation.
	We can use interned bools to avoid this cost.
*/
func boolIface(b bool) interface{} {
	if b {
		return _true
	}
	return _false
}
