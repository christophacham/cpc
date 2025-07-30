package graph

import (
	"context"
	"fmt"
	"strconv"

	"github.com/raulc0399/cpc/internal/database"
)

// Resolver is the root resolver
type Resolver struct {
	DB *database.DB
}

// Query returns the query resolver
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

// Mutation returns the mutation resolver
func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}

type queryResolver struct{ *Resolver }

// Hello is a simple hello world query
func (r *queryResolver) Hello(ctx context.Context) (string, error) {
	return "Hello from Cloud Price Compare GraphQL API!", nil
}

// Messages retrieves all messages
func (r *queryResolver) Messages(ctx context.Context) ([]*Message, error) {
	dbMessages, err := r.DB.GetMessages()
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]*Message, len(dbMessages))
	for i, msg := range dbMessages {
		messages[i] = &Message{
			ID:        strconv.Itoa(msg.ID),
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return messages, nil
}

// Providers retrieves all providers
func (r *queryResolver) Providers(ctx context.Context) ([]*Provider, error) {
	dbProviders, err := r.DB.GetProviders()
	if err != nil {
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}

	providers := make([]*Provider, len(dbProviders))
	for i, p := range dbProviders {
		providers[i] = &Provider{
			ID:        strconv.Itoa(p.ID),
			Name:      p.Name,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return providers, nil
}

// Categories retrieves all categories
func (r *queryResolver) Categories(ctx context.Context) ([]*Category, error) {
	dbCategories, err := r.DB.GetCategories()
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	categories := make([]*Category, len(dbCategories))
	for i, c := range dbCategories {
		categories[i] = &Category{
			ID:          strconv.Itoa(c.ID),
			Name:        c.Name,
			Description: c.Description,
			CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return categories, nil
}

type mutationResolver struct{ *Resolver }

// CreateMessage creates a new message
func (r *mutationResolver) CreateMessage(ctx context.Context, content string) (*Message, error) {
	msg, err := r.DB.CreateMessage(content)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return &Message{
		ID:        strconv.Itoa(msg.ID),
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}