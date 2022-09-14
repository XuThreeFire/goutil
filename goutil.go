package goutil

import "fmt"

// Go is a basic promise implementation: it wraps calls a function in a goroutine
// and returns a channel which will later return the function's return value.
func Go(f func() error) error {
	ch := make(chan error)
	go func() {
		ch <- f()
	}()
	return <-ch
}

// PanicIfErr if errorx is not empty, will panic
func PanicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

// PanicErr if errorx is not empty, will panic
func PanicErr(err error) {
	if err != nil {
		panic(err)
	}
}

// MustOK if errorx is not empty, will panic
func MustOK(err error) {
	if err != nil {
		panic(err)
	}
}

// Panicf format panic message use fmt.Sprintf
func Panicf(format string, v ...interface{}) {
	panic(fmt.Sprintf(format, v...))
}
