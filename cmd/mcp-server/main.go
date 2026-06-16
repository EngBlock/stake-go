package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	httpAddr := flag.String("http", "", "serve MCP over streamable HTTP at this address instead of stdio")
	baseURL := flag.String("base-url", "", "override Stake API base URL, primarily for tests")
	tokenFile := flag.String("token-file", "", "path to a cached Stake session token file")
	disableTokenCache := flag.Bool("no-token-cache", false, "disable reading and writing the Stake session token cache")
	enableWatchlistMutations := flag.Bool("enable-watchlist-mutations", false, "enable watchlist create/update/delete tools")
	enableOrderCancel := flag.Bool("enable-order-cancel", false, "enable pending order cancellation tools")
	flag.Parse()

	ctx := context.Background()
	auth, err := newStakeAuthFromEnv(*baseURL, *tokenFile, *disableTokenCache)
	if err != nil {
		log.Fatal(err)
	}

	server := newMCPServer(auth, serverConfig{
		EnableWatchlistMutations: *enableWatchlistMutations,
		EnableOrderCancel:        *enableOrderCancel,
	})

	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("MCP streamable HTTP listening at %s", *httpAddr)
		log.Fatal(http.ListenAndServe(*httpAddr, handler))
	}

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Printf("MCP server failed: %v", err)
	}
}
