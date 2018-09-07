package promise

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func assert(t *testing.T, expr bool, msg ...string) {
	if expr != true {
		t.Error(strings.Join(msg, " "))
	}
}

func assertEqual(t *testing.T, expected, got interface{}, msg ...string) {
	if msg == nil {
		errorMsg := fmt.Sprintf("expected: %v got: %v\n", expected, got)
		msg = append(msg, errorMsg)
	}
	assert(t, expected == got, msg...)
}

func TestPromiseBasicThen(t *testing.T) {

	p := New(func(resolve func(interface{}), reject func(error)) {
		resolve("sonla")
	})

	p.Then(func(data interface{}) interface{} {
		t.Logf("What I get is %v\n", data.(string))
		assertEqual(t, "sonla", data.(string))
		return nil
	})

	p.Await()
}

func TestPromiseBasicCatch(t *testing.T) {

	p := New(func(resolve func(interface{}), reject func(error)) {
		time.Sleep(1 * time.Second)
		reject(errors.New("sonla"))
	})

	p.Catch(func(err error) {
		t.Logf("What I get is %v\n", err)
		assertEqual(t, "sonla", err.Error())
	})

	p.Then(func(data interface{}) interface{} {
		t.Error("we don't expected to enter here")
		return nil
	}).Catch(func(err error) {
		t.Error("we don't expected to enter here")
	})

	p.Await()
}

func TestPromiseErrorInThen(t *testing.T) {

	p := New(func(resolve func(interface{}), reject func(error)) {
		resolve(map[string]string{
			"a": "apple",
			"b": "banana",
		})
	})

	p.Catch(func(err error) {
		// This will not enter because executor does not reject
		t.Error(err)
	})

	p.Then(func(data interface{}) interface{} {
		return errors.New("wow")
	}).Catch(func(err error) {
		t.Log("expect to enter error")
		assertEqual(t, err.Error(), "wow")
	})

	p.Await()
}

func TestPromiseMassiveThen(t *testing.T) {

	promises := []*Promise{}

	for i := 0; i < 20; i++ {

		p := New(func(resolve func(interface{}), reject func(error)) {
			time.Sleep(time.Second * 1)
			resolve(map[string]string{
				"a": "apple",
				"b": "banana",
			})
		})

		p.Catch(func(err error) {
			// This will not enter because executor does not reject
			t.Error(err)
		})

		p.Then(func(data interface{}) interface{} {
			return errors.New("wow")
		}).Catch(func(err error) {
			if err != nil {
				assertEqual(t, err.Error(), "wow")
			}
		})

		promises = append(promises, p)
	}

	t.Logf("promises %v\n", promises)
	All(promises...)

}

func TestPromiseThenNoErrorCatch(t *testing.T) {

	p := New(func(resolve func(interface{}), reject func(error)) {
		resolve(map[string]string{
			"a": "apple",
			"b": "banana",
		})
	})

	p.Then(func(data interface{}) interface{} {
		return nil
	}).Catch(func(err error) {
		t.Error("unexpted behavior, it should not enter here.")
	})

	p.Await()
}

func TestPromiseChainThen(t *testing.T) {

	p := New(func(resolve func(interface{}), reject func(error)) {
		resolve("Hi, ")
	})

	p.Then(func(data interface{}) interface{} {
		res := data.(string)
		res += "I am a "
		return res
	}).Then(func(data interface{}) interface{} {
		res := data.(string)
		res += "good man"

		assertEqual(t, "Hi, I am a good man", res)
		return res
	})

	p.Await()
}

func TestWaitAllPromise(t *testing.T) {

	start := time.Now()

	p1 := New(func(resolve func(interface{}), reject func(error)) {
		time.Sleep(time.Second * 1)
		resolve(1)
	})

	p1.Then(func(data interface{}) interface{} {
		time.Sleep(time.Second * 3)
		return nil
	})

	p2 := New(func(resolve func(interface{}), reject func(error)) {
		time.Sleep(time.Second * 2)
		resolve(1)
	})

	p2.Then(func(data interface{}) interface{} {
		time.Sleep(time.Second * 1)
		return nil
	})

	p3 := New(func(resolve func(interface{}), reject func(error)) {
		time.Sleep(time.Second * 1)
		resolve(1)
	})

	p3.Then(func(data interface{}) interface{} {
		time.Sleep(time.Second * 2)
		return nil
	})

	All(p1, p2, p3)

	assertEqual(t, 4, int(time.Since(start).Seconds()))
}
