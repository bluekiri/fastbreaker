fastbreaker
===========

[fastbreaker](https://github.com/bluekiri/fastbreaker) implements the [Circuit Breaker pattern](https://en.wikipedia.org/wiki/Circuit_breaker_design_pattern) in Go.

Installation
------------

```
go get github.com/bluekiri/fastbreaker
```

Usage
-----

The interface `fastbreaker.FastBreaker` is a state machine to prevent sending requests that are likely to fail.
The function `fastbreaker.New` creates a new `fastbreaker.FastBreaker`.

```go
func fastbreaker.New(configuration fastbreaker.Configuration) fastbreaker.FastBreaker
```

You can configure `fastbreaker.FastBreaker` by the struct `fastbreaker.Configuration`:

```go
type Configuration struct {
    NumBuckets      int
    BucketDuration  time.Duration
    DurationOfBreak time.Duration
    ShouldTrip      ShouldTripFunc
}
```

- `NumBuckets` is the number of buckets of the rolling window.
  If `NumBuckets` is less than 1, the `fastbreaker.DefaultNumBuckets` is used.

- `BucketDuration` is the duration (truncated to the second) of every bucket.
  If `BucketDuration` is less than 1s, the `fastbreaker.DefaultBucketDuration` is used.

- `DurationOfBreak` is the time (truncated to the second) of the open state, after which the state
  becomes half-open.
  If `DurationOfBreak` is less than 1s, the `fastbreaker.DefaultDurationOfBreak` is used.

- `ShouldTrip` is called whenever a request fails in the closed state with the number of executions
  and the number of failures.
  If `ShouldTrip` returns true, `fastbreaker.FastBreaker` state becomes open.
  If `ShouldTrip` is `nil`, `fastbreaker.DefaultShouldTrip` is used.
  `fastbreaker.DefaultShouldTrip` returns true when the number of executions is greater than or equal
  to 10 and at least half the number of executions have failed.

Example
-------

```go
var cb fastbreaker.FastBreaker

func Get(url string) ([]byte, error) {
	feedback, err := cb.Allow()
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		feedback(false)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		feedback(false)
		return nil, err
	}

	feedback(true)
	return body, nil
}
```

License
-------

The MIT License (MIT)

See [LICENSE](https://github.com/bluekiri/fastbreaker/blob/master/LICENSE) for details.
