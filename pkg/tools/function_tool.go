// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"context"
	"fmt"
	"reflect"
)

// FunctionTool wraps a Go function as a tool that can be used by agents.
type FunctionTool struct {
	*BaseTool
	function     interface{}
	takesCtx     bool
	takesToolCtx bool
}

// FunctionToolConfig contains configuration options for a FunctionTool.
type FunctionToolConfig struct {
	// Name is the name of the tool. If empty, the function name will be used.
	Name string

	// Description describes what the tool does.
	Description string

	// InputSchema defines the expected input parameters.
	InputSchema ParameterSchema

	// OutputSchema defines the expected output parameters.
	OutputSchema map[string]ParameterSchema

	// IsLongRunning indicates if the tool takes a long time to execute.
	IsLongRunning bool
}

// NewFunctionTool creates a tool that wraps a Go function.
// The function can have any signature, but must return either a single value
// or a value and an error. The tool will attempt to convert input parameters
// to the function's parameter types based on name matching.
func NewFunctionTool(fn interface{}, config FunctionToolConfig) (*LlmToolAdaptor, error) {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected a function, got %T", fn)
	}

	// Use function name as default if name not specified
	name := config.Name
	if name == "" {
		name = getFunctionName(fn)
	}

	// Check if function accepts context and/or ToolContext
	takesCtx := false
	takesToolCtx := false

	for i := 0; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		// Check for context.Context
		if paramType.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			takesCtx = true
		}
		// Check for *ToolContext
		if paramType == reflect.TypeOf(&ToolContext{}) {
			takesToolCtx = true
		}
	}

	// Verify the function returns at least one value
	if fnType.NumOut() == 0 {
		return nil, fmt.Errorf("function must return at least one value")
	}

	// The second return value, if present, should be an error
	if fnType.NumOut() > 1 && !fnType.Out(fnType.NumOut()-1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, fmt.Errorf("last return value must be error if function has multiple return values")
	}

	// Create the function tool
	functionTool := &FunctionTool{
		function:     fn,
		takesCtx:     takesCtx,
		takesToolCtx: takesToolCtx,
	}

	// Create base tool with execute function that delegates to our function
	baseTool := &BaseTool{
		name:        name,
		description: config.Description,
		schema: ToolSchema{
			Input:  config.InputSchema,
			Output: config.OutputSchema,
		},
		executeFn: functionTool.execute,
	}

	functionTool.BaseTool = baseTool

	// Create and return the tool adaptor
	return NewLlmToolAdaptor(baseTool, config.IsLongRunning), nil
}

// execute runs the wrapped function with the given input arguments
func (ft *FunctionTool) execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	fnValue := reflect.ValueOf(ft.function)
	fnType := fnValue.Type()

	// Prepare arguments
	args := make([]reflect.Value, 0, fnType.NumIn())

	// Handle special arguments (context and toolContext) first
	if ft.takesCtx {
		args = append(args, reflect.ValueOf(ctx))
	}

	// The rest are from the input map based on parameter names
	for i := len(args); i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)

		// Check if parameter is ToolContext
		if paramType == reflect.TypeOf(&ToolContext{}) && ft.takesToolCtx {
			// This would normally come from the execution context
			// For now, create an empty one, this will be filled in by the LlmToolAdaptor
			args = append(args, reflect.ValueOf(&ToolContext{}))
			continue
		}

		// Get parameter name
		paramName := fnType.In(i).Name()
		if paramName == "" {
			paramName = fmt.Sprintf("param%d", i)
		}

		// Look for parameter in input
		value, exists := input[paramName]
		if !exists {
			// If parameter is missing, see if we can use the default value
			if fnType.IsVariadic() && i == fnType.NumIn()-1 {
				// For variadic functions, provide an empty slice
				args = append(args, reflect.MakeSlice(paramType, 0, 0))
			} else {
				// Otherwise, use zero value
				args = append(args, reflect.Zero(paramType))
			}
			continue
		}

		// Convert input value to parameter type
		convertedValue, err := convertValueToType(value, paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to convert parameter %s: %v", paramName, err)
		}

		args = append(args, convertedValue)
	}

	// Call the function
	results := fnValue.Call(args)

	// Handle the results
	if len(results) == 0 {
		return map[string]interface{}{}, nil
	}

	// Check for error (should be the last result)
	if len(results) > 1 {
		errVal := results[len(results)-1]
		if !errVal.IsNil() {
			return nil, errVal.Interface().(error)
		}
		results = results[:len(results)-1] // Remove error result
	}

	// Convert first result to map[string]interface{}
	result := results[0].Interface()

	// If result is already a map, return it
	if resultMap, ok := result.(map[string]interface{}); ok {
		return resultMap, nil
	}

	// Otherwise, wrap the result in a map
	return map[string]interface{}{
		"result": result,
	}, nil
}

// getFunctionName attempts to get a meaningful name for a function
func getFunctionName(fn interface{}) string {
	return fmt.Sprintf("function_%p", fn)
}

// convertValueToType converts a value to the expected type
func convertValueToType(value interface{}, targetType reflect.Type) (reflect.Value, error) {
	// Handle nil
	if value == nil {
		return reflect.Zero(targetType), nil
	}

	// Get value's type
	valueType := reflect.TypeOf(value)

	// If types are already compatible, use direct conversion
	if valueType.AssignableTo(targetType) {
		return reflect.ValueOf(value), nil
	}

	// Handle basic type conversions
	switch targetType.Kind() {
	case reflect.String:
		return reflect.ValueOf(fmt.Sprintf("%v", value)), nil

	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			return reflect.ValueOf(v), nil
		case string:
			if v == "true" {
				return reflect.ValueOf(true), nil
			} else if v == "false" {
				return reflect.ValueOf(false), nil
			}
		}
		return reflect.Value{}, fmt.Errorf("cannot convert %v to bool", value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int:
			return reflect.ValueOf(v).Convert(targetType), nil
		case int64:
			return reflect.ValueOf(v).Convert(targetType), nil
		case float64:
			return reflect.ValueOf(int64(v)).Convert(targetType), nil
		case string:
			// Would need parsing, but skipping for simplicity
		}
		return reflect.Value{}, fmt.Errorf("cannot convert %v to int", value)

	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			return reflect.ValueOf(v).Convert(targetType), nil
		case int:
			return reflect.ValueOf(float64(v)).Convert(targetType), nil
		}
		return reflect.Value{}, fmt.Errorf("cannot convert %v to float", value)

	case reflect.Map, reflect.Slice:
		// Would need more complex conversion, but skipping for simplicity
	}

	return reflect.Value{}, fmt.Errorf("unsupported type conversion from %T to %v", value, targetType)
}
