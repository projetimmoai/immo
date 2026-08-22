package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// ticketRow est la représentation JSON d'une ligne de la table ticket.
type ticketRow struct {
	ID              int64     `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	Reference       string    `json:"reference"`
	ActionID        int64     `json:"action_id"`
	SousActionID    *int64    `json:"sous_action_id"`
	StatutID        int64     `json:"statut_id"`
	SourceID        int64     `json:"source_id"`
	CoproprieteID   int64     `json:"copropriete_id"`
	LotID           *int64    `json:"lot_id"`
	ParentID        *int64    `json:"parent_id"`
	AssigneA        *int64    `json:"assigne_a"`
	CreePar         *int64    `json:"cree_par"`
	DateDeclaration time.Time `json:"date_declaration"`
}

func (r ticketRow) toDomain() *domain.Ticket {
	return &domain.Ticket{
		ID:              r.ID,
		CreatedAt:       r.CreatedAt,
		Reference:       r.Reference,
		ActionID:        r.ActionID,
		SousActionID:    r.SousActionID,
		StatutID:        r.StatutID,
		SourceID:        r.SourceID,
		CoproprieteID:   r.CoproprieteID,
		LotID:           r.LotID,
		ParentID:        r.ParentID,
		AssigneA:        r.AssigneA,
		CreePar:         r.CreePar,
		DateDeclaration: r.DateDeclaration,
	}
}

// UpdateTicketStatut change le statut d'un Ticket existant (cf.
// domain.TicketStatut* pour le vocabulaire de cycle de vie partagé par tous
// les types de ticket).
func (c *Client) UpdateTicketStatut(ctx context.Context, ticketID, statutID int64) error {
	path := fmt.Sprintf("/ticket?id=eq.%d", ticketID)
	payload := map[string]any{"statut_id": statutID}
	if err := c.do(ctx, http.MethodPatch, path, payload, nil); err != nil {
		return fmt.Errorf("repository: mise à jour du statut du ticket id=%d (statut_id=%d): %w", ticketID, statutID, err)
	}
	return nil
}

// FindTicketByID retrouve un Ticket par son ID. Retourne (nil, nil), sans
// erreur, si aucun ticket ne correspond.
func (c *Client) FindTicketByID(ctx context.Context, id int64) (*domain.Ticket, error) {
	var rows []ticketRow
	path := fmt.Sprintf("/ticket?select=*&id=eq.%d&limit=1", id)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche ticket id=%d: %w", id, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}
