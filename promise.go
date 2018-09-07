package promise

import (
	"fmt"
)

const (
	pending = iota
	fulfilled
	rejected
)

// Promise mimics the javascript's promise
type Promise struct {

	// state pending 0, fullfilled 1, rejected 2
	state int

	executor       func(resolve func(interface{}), reject func(error))
	resolveChannel chan interface{}
	rejectChannel  chan error
	chain          *Promise

	isLast bool
	// a flag to note whether this is the last promise or not
	// we can not judge by chain is nil because the chain is assigned
	// to next promise when the channel received data which is dynamically
	// decided.

	done chan struct{} // the signal for promise is done
}

// New instantiate a new promise object
func New(executor func(resolve func(interface{}), reject func(error))) *Promise {

	promise := &Promise{
		state:          pending,
		executor:       executor,
		resolveChannel: make(chan interface{}, 1),
		rejectChannel:  make(chan error, 1),
		chain:          nil,
		isLast:         true,
		done:           make(chan struct{}),
	}

	go func() {

		// catch exception error happen in the executor
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					promise.reject(err)
				} else {
					promise.reject(fmt.Errorf("%v", r))
				}
			}
		}()

		promise.executor(promise.resolve, promise.reject)

	}()

	return promise
}

func (p *Promise) reject(err error) {
	if p.state != pending {
		return
	}

	p.state = rejected
	p.rejectChannel <- err
}

// Resolve return a new Promise as a resolved promise
func (p *Promise) Resolve(value interface{}) *Promise {
	return New(func(resolve func(interface{}), reject func(error)) {
		resolve(value)
	})
}

func (p *Promise) resolve(value interface{}) {
	if p.state != pending {
		return
	}

	p.state = fulfilled
	p.resolveChannel <- value
}

// Then accept the data from the resolver and return a new promise
func (p *Promise) Then(fulfill func(data interface{}) interface{}) *Promise {
	// I don't know it's a good idea to return an interface or not..
	// if we define fulfill use func(data interface{}) interface then
	// we can make fulfill to return error or a new promise with the value it returned
	var result *Promise

	result = New(func(resolve func(interface{}), reject func(error)) {

		select {
		case resolution := <-p.resolveChannel:
			p.chain = result
			response := fulfill(resolution)

			if err, ok := response.(error); ok && err != nil {
				reject(err)
			} else {
				resolve(response)
				// a tricky way to enable sending resolve and rejeuct channel
				p.chain.state = pending
				reject(nil)
			}

			p.done <- struct{}{}
		}
	})

	p.isLast = false
	return result
}

// Catch accept the data from the rejector and return a new promise
func (p *Promise) Catch(rejected func(err error)) *Promise {

	var result *Promise

	result = New(func(resolve func(interface{}), reject func(error)) {

		select {

		case rejection := <-p.rejectChannel:
			p.chain = result
			if rejection != nil {
				rejected(rejection)
			}
			resolve(true)
			p.done <- struct{}{}
		}

	})

	p.isLast = false
	return result
}

// Await waits the promise to complete
func (p *Promise) Await() {

	for p != nil && !p.isLast {

		_, opened := <-p.done
		if opened {
			close(p.done)
		}
		p = p.chain

		if p.chain == nil {
			break
		}

	}

}

// All waits the all promises to complete
func All(promises ...*Promise) {
	for _, p := range promises {
		p.Await()
	}
}
