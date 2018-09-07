# Go-promise

This is an experimental implementation of [javascript's promise](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise) in golang. Why implement this? Actually, golang is not suitable to implement generic purpose library and it has a good async funtion with `select`, `go routine`, `channel`. I have to say those tools are powerful. In javascript, you have to use `promise` or `async await` to achieve the same effect. However, I think `promise` is a asynchronous pattern not a basic async funtion so I am interested at implementation it in golang. That's why I create this repo.

## Example

a simple example list in the following.

```golang

	p := New(func(resolve func(interface{}), reject func(error)) {
		resolve("sonla")
	})

	p.Then(func(data interface{}) interface{} {
		t.Logf("What I get is %v\n", data.(string))
		assertEqual(t, "sonla", data.(string))
		return nil
	})

	p.Await()

```

I change the behavior of promise.then in my implementation because golang doesn't support optional arguement. If we want the function then to accept onSuccess and onError, we need to define function signature of then to accept both function. We can not ignore one of them when we call it. For example,

```js

var p = new Promise((resolve, reject) => {
   resolve("hi")
}).then((value) => {
   console.log(value)
})

// in js we can only accept onSuccess but in golang..
// it will be like the following
// p.Then(func(data interface{}), interface{}, func(err error){
// })

```

I feel uncomfortable with that so I change the behavior a little :p
