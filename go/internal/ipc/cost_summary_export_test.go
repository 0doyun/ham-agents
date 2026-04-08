package ipc

import "context"

// DispatchForTest exposes the package-private dispatch path so the
// cost.summary handler can be exercised without spinning up a real unix
// socket. Test-only — do not call from production code.
func DispatchForTest(server *Server, ctx context.Context, request Request) (Response, error) {
	return server.dispatch(ctx, request)
}
