// Package client provides the core Langfuse client implementation.
//
// This package contains the main Client struct and its methods for
// interacting with the Langfuse API. It handles:
//   - Event batching and ingestion
//   - Background processing
//   - Lifecycle management
//   - Graceful shutdown
//
// Most users should import the root langfuse package which provides
// a facade over this package with additional convenience features.
//
// Example:
//
//	import "github.com/jdziat/langfuse-go/pkg/client"
//
//	c, err := client.New("pk-lf-xxx", "sk-lf-xxx")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer c.Shutdown(context.Background())
package client
