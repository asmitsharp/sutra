package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/ashmitsharp/sutra/mcp"
	"github.com/ashmitsharp/sutra/mcp/sse"
)

func main() {
	transport := sse.New()

	caps := mcp.ServerCapabilities{
		Tools: &mcp.ToolsCapability{
			ListChanged: false,
		},
	}

	session := mcp.NewSession("inspect-session", transport, caps)

	router := mcp.NewRouter()

	router.Handle("tools/list", func(ctx context.Context, s *mcp.Session, params json.RawMessage) (any, error) {
		return map[string]any{"tools": []any{}}, nil
	})

	ctx := context.Background()
	go func() {
		if err := router.Serve(ctx, session); err != nil {
			log.Printf("serve loop ended: %v", err)
		}
	}()

	log.Println("listening on :8080")
	http.ListenAndServe(":8080", transport.Handler())

}
