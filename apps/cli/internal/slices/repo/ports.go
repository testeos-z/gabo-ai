package repo

import "context"

// API calls the GitHub REST API as the owner. body, when non-nil, is a JSON
// document sent on stdin; the response body is returned.
type API interface {
	Call(ctx context.Context, method, path string, body []byte) ([]byte, error)
}
