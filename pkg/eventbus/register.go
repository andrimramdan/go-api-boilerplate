package eventbus

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/vardius/gollback"
	"google.golang.org/grpc"

	grpcutils "github.com/vardius/go-api-boilerplate/pkg/grpc"
)

// RegisterGRPCHandlers registers event handlers for topics
// will panic after timeout if unable to register handlers
func RegisterGRPCHandlers(serviceName string, conn *grpc.ClientConn, eventBus EventBus, topicToHandlerMap map[string]EventHandler, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Will retry infinitely until timeouts by context
	_, err := gollback.Retry(ctx, 0, func(ctx context.Context) (interface{}, error) {
		if !grpcutils.IsConnectionServing(ctx, serviceName, conn) {
			return nil, fmt.Errorf(" %s gRPC connection is not serving", "pushpull")
		}

		for topic, handler := range topicToHandlerMap {
			// Will resubscribe to handler on error infinitely
			go func(topic string, handler EventHandler) {
				// this goroutine runs independently to request's goroutine,
				// therefor recover middleware will not recover from panic to prevent crash
				defer func() {
					if r := recover(); r != nil {
						log.Fatalf("[EventHandler] Recovered in %v\n%s\n", r, debug.Stack())
					}
				}()

				_, _ = gollback.Retry(context.Background(), 0, func(ctx context.Context) (interface{}, error) {
					err := eventBus.Subscribe(ctx, topic, handler)

					return nil, fmt.Errorf("EventHandler %s unsubscribed (%v)", topic, err)
				})
			}(topic, handler)
		}

		return nil, nil
	})

	if err != nil {
		panic(err)
	}
}
