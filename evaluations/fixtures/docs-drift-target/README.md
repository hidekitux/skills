# retry

Run an operation until it succeeds.

## Usage

```go
err := retry.Run(3, func() error {
	return send(request)
})
```

`Run` tries the operation three times by default and returns the last error.

## Development

Run the tests with `make test`.
