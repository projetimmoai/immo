package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// actionRow est la représentation JSON d'une ligne de la table action.
type actionRow struct {
	ID          int64     `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
}

func (r actionRow) toDomain() *domain.Action {
	return &domain.Action{ID: r.ID, CreatedAt: r.CreatedAt, Description: r.Description}
}

// ListActions retourne toutes les actions possibles (table action).
func (c *Client) ListActions(ctx context.Context) ([]*domain.Action, error) {
	var rows []actionRow
	if err := c.do(ctx, http.MethodGet, "/action?select=*", nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: listage action: %w", err)
	}
	result := make([]*domain.Action, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.toDomain())
	}
	return result, nil
}
