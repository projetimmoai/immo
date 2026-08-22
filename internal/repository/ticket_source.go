package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// ticketSourceRow est la représentation JSON d'une ligne de la table ticket_source.
type ticketSourceRow struct {
	ID                 int64      `json:"id"`
	CreatedAt          time.Time  `json:"created_at"`
	TypeID             int64      `json:"type_id"`
	DateReception      time.Time  `json:"date_reception"`
	PersonneID         *int64     `json:"personne_id"`
	CoproprieteID      *int64     `json:"copropriete_id"`
	LotID              *int64     `json:"lot_id"`
	StatutTraitementID int64      `json:"statut_traitement_id"`
	TraiteLe           *time.Time `json:"traite_le"`
	TraitePar          *int64     `json:"traite_par"`
	ErreurTraitement   *string    `json:"erreur_traitement"`
}

func (r ticketSourceRow) toDomain() *domain.TicketSource {
	return &domain.TicketSource{
		ID:                 r.ID,
		CreatedAt:          r.CreatedAt,
		TypeID:             r.TypeID,
		DateReception:      r.DateReception,
		PersonneID:         r.PersonneID,
		CoproprieteID:      r.CoproprieteID,
		LotID:              r.LotID,
		StatutTraitementID: r.StatutTraitementID,
		TraiteLe:           r.TraiteLe,
		TraitePar:          r.TraitePar,
		ErreurTraitement:   r.ErreurTraitement,
	}
}

// ticketSourceInsert est la charge utile d'insertion d'une TicketSource :
// uniquement les colonnes fournies par l'appelant (id/created_at sont
// générés par la base).
type ticketSourceInsert struct {
	TypeID             int64      `json:"type_id"`
	DateReception      time.Time  `json:"date_reception"`
	PersonneID         *int64     `json:"personne_id,omitempty"`
	CoproprieteID      *int64     `json:"copropriete_id,omitempty"`
	LotID              *int64     `json:"lot_id,omitempty"`
	StatutTraitementID int64      `json:"statut_traitement_id"`
	TraiteLe           *time.Time `json:"traite_le,omitempty"`
	TraitePar          *int64     `json:"traite_par,omitempty"`
	ErreurTraitement   *string    `json:"erreur_traitement,omitempty"`
}

// InsertTicketSource insère une nouvelle TicketSource et retourne la ligne
// créée (avec son ID généré par la base). s.StatutTraitementID doit être
// renseigné explicitement par l'appelant (pas de valeur par défaut en
// base) — cf. TicketSourceStatutTraitementID pour le résoudre à partir
// d'une description (voir les constantes domain.TicketSourceStatut*).
func (c *Client) InsertTicketSource(ctx context.Context, s *domain.TicketSource) (*domain.TicketSource, error) {
	payload := []ticketSourceInsert{{
		TypeID:             s.TypeID,
		DateReception:      s.DateReception,
		PersonneID:         s.PersonneID,
		CoproprieteID:      s.CoproprieteID,
		LotID:              s.LotID,
		StatutTraitementID: s.StatutTraitementID,
		TraiteLe:           s.TraiteLe,
		TraitePar:          s.TraitePar,
		ErreurTraitement:   s.ErreurTraitement,
	}}
	var rows []ticketSourceRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/ticket_source", payload, "return=representation", &rows); err != nil {
		return nil, fmt.Errorf("repository: insertion ticket_source (type_id=%d): %w", s.TypeID, err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion ticket_source: %d ligne(s) retournée(s), 1 attendue", len(rows))
	}
	return rows[0].toDomain(), nil
}

// DeleteTicketSource supprime une TicketSource par son ID (utilisé
// notamment par les tests d'intégration pour nettoyer après eux). Les
// tables détail (email, message_application, message_telephonique)
// associées sont supprimées en cascade (ON DELETE CASCADE en base).
func (c *Client) DeleteTicketSource(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/ticket_source?id=eq.%d", id)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("repository: suppression ticket_source id=%d: %w", id, err)
	}
	return nil
}
