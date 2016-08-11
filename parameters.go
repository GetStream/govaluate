package govaluate

import (
	"errors"
)

// Parameters is a collection of named parameters that are accessible via the
// Get method.
type Parameters interface {
	// Get gets the parameter of the given name
	Get(name string) (interface{}, error)
}

// MapParameters is an implementation of the Parameters interface using a map.
type MapParameters map[string]interface{}

// Get implemetns the method from Parameters
func (p MapParameters) Get(name string) (interface{}, error) {

	value, found := p[name]

	if(!found) {
		errorMessage := "No parameter '" + name + "' found."
		return nil, errors.New(errorMessage)
	}

	return value, nil
}
