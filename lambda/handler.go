package lambda

// base 来自aws-lambda-go
// guonaihong:删除了一些用不到的参数,
// 修改reflectHandler 参数返回error

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"golang.org/x/net/context"
)

type Handler interface {
	Invoke(ctx context.Context, payload []byte) ([]byte, error)
}

type bytesHandlerFunc func(context.Context, []byte) ([]byte, error)

func (h bytesHandlerFunc) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	return h(ctx, payload)
}

//	func ()
//	func () error
//	func (TIn) error
//	func () (TOut, error)
//	func (TIn) (TOut, error)
//	func (context.Context) error
//	func (context.Context, TIn) error
//	func (context.Context) (TOut, error)
//	func (context.Context, TIn) (TOut, error)

// 检测函数形参
func validateArguments(handler reflect.Type) (bool, error) {
	handlerTakesContext := false
	// 最多两个形参，超过报错
	if handler.NumIn() > 2 {
		return false, fmt.Errorf("handlers may not take more than two arguments, but handler takes %d", handler.NumIn())
	} else if handler.NumIn() > 0 {
		// 处理参数是1, 2 的情况
		// 拿到context的接口
		contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
		// 拿到第一个参数的类型
		argumentType := handler.In(0)
		// 第一个参数有没有实现接口context.Context的接口
		handlerTakesContext = argumentType.Implements(contextType)
		if handler.NumIn() > 1 && !handlerTakesContext {
			return false, fmt.Errorf("handler takes two arguments, but the first is not Context. got %s", argumentType.Kind())
		}
	}

	// 0, 1, 2参数在这返回
	return handlerTakesContext, nil
}

// 检测结果
func validateReturns(handler reflect.Type) error {
	errorType := reflect.TypeOf((*error)(nil)).Elem()

	switch n := handler.NumOut(); {
	// 返回值超过2个的直接报错
	case n > 2:
		return fmt.Errorf("handler may not return more than two values")
		// 第2个参数不是error的直接报错
	case n > 1:
		if !handler.Out(1).Implements(errorType) {
			return fmt.Errorf("handler returns two values, but the second does not implement error")
		}
	case n == 1:
		// 只有一个参数的，并且不是error类型的直接报错
		if !handler.Out(0).Implements(errorType) {
			return fmt.Errorf("handler returns a single value, but it does not implement error")
		}
	}

	return nil
}

func reflectHandler(handlerFunc interface{}) (Handler, error) {
	if handlerFunc == nil {
		return nil, errors.New("handler is nil")
	}

	if handler, ok := handlerFunc.(Handler); ok {
		return handler, nil
	}

	handler := reflect.ValueOf(handlerFunc)
	handlerType := reflect.TypeOf(handlerFunc)
	if handlerType.Kind() != reflect.Func {
		return nil, fmt.Errorf("handler kind %s is not %s", handlerType.Kind(), reflect.Func)
	}

	takesContext, err := validateArguments(handlerType)
	if err != nil {
		return nil, err
	}

	if err := validateReturns(handlerType); err != nil {
		return nil, err
	}

	return bytesHandlerFunc(func(ctx context.Context, payload []byte) ([]byte, error) {
		in := bytes.NewBuffer(payload)
		out := bytes.NewBuffer(nil)
		decoder := json.NewDecoder(in)
		encoder := json.NewEncoder(out)

		// construct arguments
		var args []reflect.Value
		if takesContext {
			args = append(args, reflect.ValueOf(ctx))
		}

		if (handlerType.NumIn() == 1 && !takesContext) || handlerType.NumIn() == 2 {
			eventType := handlerType.In(handlerType.NumIn() - 1)
			event := reflect.New(eventType)
			if err := decoder.Decode(event.Interface()); err != nil {
				return nil, err
			}
			args = append(args, event.Elem())
		}

		response := handler.Call(args)

		// return the error, if any
		if len(response) > 0 {
			if errVal, ok := response[len(response)-1].Interface().(error); ok && errVal != nil {
				return nil, errVal
			}
		}
		// set the response value, if any
		var val interface{}
		if len(response) > 1 {
			val = response[0].Interface()
		}

		if err := encoder.Encode(val); err != nil {
			return nil, err
		}

		responseBytes := out.Bytes()

		return responseBytes, nil
	}), nil
}
