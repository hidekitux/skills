package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// apiTimeout bounds one GitHub REST request. It matches the timeout the
// signature check used before the provider boundary existed.
const apiTimeout = 30 * time.Second

// GitHubAPI is the GitHub REST port. It is separate from the GitHub command
// line port because its authority is different: a request carries a token the
// caller supplies, while the gh command line uses the authentication already
// present in the environment.
type GitHubAPI interface {
	// Get reads one API path relative to the API root and returns the
	// response body. The adapter sends the token and never returns it.
	Get(ctx context.Context, path, token string) ([]byte, error)
}

// NewGitHubAPI returns the REST adapter. An empty root uses the public API.
func NewGitHubAPI(root string) GitHubAPI {
	if root == "" {
		root = "https://api.github.com"
	}
	return &restAdapter{root: root, client: &http.Client{Timeout: apiTimeout}}
}

type restAdapter struct {
	root   string
	client *http.Client
}

func (a *restAdapter) Get(ctx context.Context, path, token string) ([]byte, error) {
	operation := "GET " + path
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, a.root+path, nil)
	if err != nil {
		return nil, &Error{Port: PortAPI, Operation: operation, Kind: KindFailure, Err: err}
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := a.client.Do(request)
	if err != nil {
		return nil, &Error{
			Port:      PortAPI,
			Operation: operation,
			Kind:      requestKind(ctx, err),
			Detail:    Redact(err.Error()),
			Err:       err,
		}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, &Error{
			Port:      PortAPI,
			Operation: operation,
			Kind:      requestKind(ctx, err),
			Detail:    Redact(err.Error()),
			Err:       err,
		}
	}
	if response.StatusCode >= 400 {
		return body, &Error{
			Port:      PortAPI,
			Operation: operation,
			Kind:      KindFailure,
			ExitCode:  response.StatusCode,
			Detail:    fmt.Sprintf("HTTP %d", response.StatusCode),
		}
	}
	return body, nil
}

// requestKind classifies a transport failure. A cancelled context is an
// interruption and an expired deadline is a timeout, matching the process
// classification so a caller reads both ports the same way.
func requestKind(ctx context.Context, err error) Kind {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return KindInterrupted
	case errors.Is(ctx.Err(), context.DeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		return KindTimeout
	}
	var timeout interface{ Timeout() bool }
	if errors.As(err, &timeout) && timeout.Timeout() {
		return KindTimeout
	}
	return KindFailure
}
