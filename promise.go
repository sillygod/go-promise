package promise

import (
	"context"
	"fmt"
	"reflect"
)

const (
	pending = iota
	fulfilled
	rejected
)

// Promise mimics the javascript's promise
// currently, use single linked list to implement the chain
// of promise.
// TODO maybe consider to use double linked list instead
type Promise struct {

	// state pending 0, fulfilled 1, rejected 2
	state int

	executor       func(resolve func(interface{}), reject func(error))
	resolveChannel chan interface{}
	rejectChannel  chan error
	result         interface{} // store the result when resolve, current reject not consider
	chain          *Promise

	isLast bool
	// a flag to note whether this is the last promise or not
	// we can not judge by chain is nil because the chain is assigned
	// to next promise when the channel received data which is dynamically
	// decided.

	lastResult     interface{}
	lastResultChan chan struct{}
	// lastResult and lastResultChan are used to store resutl of the last promise in the chain
	// and send the signal to channel for race function

	ctx    context.Context // used for cleaning leak goroutine
	endSig context.CancelFunc

	done chan struct{} // the signal for promise is done
}

func newWithContext(ctx context.Context, endSig context.CancelFunc, executor func(resolve func(interface{}), reject func(error))) *Promise {

	promise := &Promise{
		state:          pending,
		executor:       executor,
		resolveChannel: make(chan interface{}, 1),
		rejectChannel:  make(chan error, 1),
		chain:          nil,
		ctx:            ctx,
		endSig:         endSig,
		isLast:         true,
		lastResultChan: make(chan struct{}, 1),
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

				// promise.done <- struct{}{}
			}
		}()

		promise.executor(promise.resolve, promise.reject)

	}()

	return promise
}

// New instantiate a new promise object
func New(executor func(resolve func(interface{}), reject func(error))) *Promise {
	ctx, cancel := context.WithCancel(context.Background())
	return newWithContext(ctx, cancel, executor)
}

// Reject return a new Promise as a reject promise
func (p *Promise) Reject(err error) *Promise {
	return New(func(resolve func(interface{}), reject func(error)) {
		reject(err)
	})
}

func (p *Promise) reject(err error) {
	if p.state != pending {
		return
	}

	p.state = rejected
	p.rejectChannel <- err
}

func (p *Promise) resetState() {
	p.state = pending
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

	result = newWithContext(p.ctx, p.endSig, func(resolve func(interface{}), reject func(error)) {

		select {
		case resolution := <-p.resolveChannel:
			func() {
				defer func() {
					p.done <- struct{}{}
				}()

				p.chain = result
				response := fulfill(resolution)
				p.result = response

				if err, ok := response.(error); ok && err != nil {
					reject(err)
				} else {
					resolve(response)
					result.resetState()
					reject(nil)
				}

			}()
		case <-p.ctx.Done():
		}
	})

	p.isLast = false
	return result
}

// Catch accept the data from the rejector and return a new promise
func (p *Promise) Catch(rejected func(err error)) *Promise {

	var result *Promise

	result = newWithContext(p.ctx, p.endSig, func(resolve func(interface{}), reject func(error)) {

		select {

		case rejection := <-p.rejectChannel:

			func() {
				defer func() {
					if rejection != nil {
						// it seems that it's legal to chain with then after a catch
						// even there is no error happen.
						// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Using_promises#Chaining_after_a_catch
						result.resolve(true)
					}
					p.done <- struct{}{}
				}()

				p.chain = result

				if rejection != nil {
					rejected(rejection)
				}

			}()
		case <-p.ctx.Done():
		}

	})

	p.isLast = false
	return result
}

// completeChan return the done channel of the last promise
func (p *Promise) completeChan() chan struct{} {

	var pre *Promise

	for p != nil {
		pre = p
		p = p.chain
	}

	return pre.done
}

// Await wait a promise to complete
func Await(p *Promise) *Promise {

	ptr := p
	pre := p
	if ptr.isLast {
		// promise is not called by any method like Then or Catch

		ptr.Then(func(data interface{}) interface{} {
			return data
		})
	}

	for ptr != nil && !ptr.isLast {
		_, opened := <-ptr.done
		if opened {
			close(ptr.done)
		}

		pre = ptr
		ptr = ptr.chain

	}

	p.lastResult = pre.result

	go func() {
		p.lastResultChan <- struct{}{}
	}()
	// make the unreached promise closed, this will be used to judge
	// the completion of promise chain

	p.endSig()
	// there maybe an issue with the leak of goroutines so use context
	// to control process flow. Sending the end signal to the unreached
	// promise to make them done.

	return pre

}

// All return a single promise that settled all of the promises
func All(promises ...*Promise) *Promise {

	result := make([]interface{}, 0, len(promises))

	for _, p := range promises {
		Await(p)
		result = append(result, p.lastResult)
	}

	p := New(func(resolve func(interface{}), reject func(error)) {
		resolve(result)
	})

	return p
}

// Race return a promise that resolves or rejects as soon as
// one of the promises resolves or rejects
func Race(promises ...*Promise) *Promise {

	cases := make([]reflect.SelectCase, 0, len(promises))

	//https://stackoverflow.com/questions/19992334/how-to-listen-to-n-channels-dynamic-select-statement
	// maybe we can learn some idea from this post

	for _, p := range promises {

		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(p.lastResultChan),
		})

		go func(ip *Promise) {
			Await(ip)
		}(p)

	}

	for {
		chosen, _, ok := reflect.Select(cases)
		// how about remains, it's good to stop other cases's process
		// to reduce the resource spend
		if ok {
			p := New(func(resolve func(interface{}), reject func(error)) {
				resolve(promises[chosen].lastResult)
			})

			return p
		}
	}

}
