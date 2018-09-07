# Go-promise

This is an experimental implementation of [javascript's promise](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise) in golang. Why implement this? Actually, golang is not suitable to implement generic purpose library and it has a good async funtion with `select`, `go routine`, `channel`. I have to say those tools are powerful. In javascript, you have to use `promise` or `async await` to achieve the same effect. However, I think `promise` is a asynchronous pattern not a basic async funtion so I am interested at implementation it in golang. That's why I create this repo.

