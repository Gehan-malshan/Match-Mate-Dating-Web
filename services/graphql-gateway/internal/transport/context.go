package transport

import (
	"context"
	"net/http"
)

type requestKey struct{}
type writerKey struct{}

func WithHTTP(ctx context.Context, request *http.Request, writer http.ResponseWriter) context.Context {
	ctx = context.WithValue(ctx, requestKey{}, request)
	return context.WithValue(ctx, writerKey{}, writer)
}

func Request(ctx context.Context) *http.Request {
	request, _ := ctx.Value(requestKey{}).(*http.Request)
	return request
}

func Writer(ctx context.Context) http.ResponseWriter {
	writer, _ := ctx.Value(writerKey{}).(http.ResponseWriter)
	return writer
}
