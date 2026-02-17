package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func New() *mcp.Server {
	return mcp.NewServer(
		&mcp.Implementation{
			Name:    "rest-api-mcp",
			Version: "0.1.0",
		},
		nil,
	)
}

func Run(server *mcp.Server) error {
	return server.Run(context.Background(), &mcp.StdioTransport{})
}
