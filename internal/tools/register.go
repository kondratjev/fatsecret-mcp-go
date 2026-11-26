package tools

import (
	"context"

	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// bind adapts a plain handler func to the SDK handler signature.
func bind[In, Out any](s *api.Server, h func(*api.Server, context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		return h(s, ctx, req, in)
	}
}

// RegisterAll registers every FatSecret tool on the server.
func RegisterAll(s *api.Server) {
	registerPublicFoodTools(s)
	registerFeedbackTools(s)
	registerRecognitionTools(s)
	registerRecipeTools(s)
	registerReferenceTools(s)
	registerFoodDiaryTools(s)
	registerFavoriteTools(s)
	registerSavedMealTools(s)
	registerWeightTools(s)
	registerExerciseTools(s)
	registerProfileTools(s)
	registerAuthTools(s)
}
