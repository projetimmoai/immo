package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// reclamationRow est la représentation JSON d'une ligne de la table
// reclamation.
type reclamationRow struct {
	ID            int64      `json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	TicketID      int64      `json:"ticket_id"`
	PrestataireID int64      `json:"prestataire_id"`
	Texte         string     `json:"texte"`
	StatutID      int64      `json:"statut_id"`
	DateEnvoi     *time.Time `json:"date_envoi"`
	DateReponse   *time.Time `json:"date_reponse"`
	ReponseTexte  *string    `json:"reponse_texte"`
	CreePar       *int64     `json:"cree_par"`
}

func (r reclamationRow) toDomain() *domain.Reclamation {
	return &domain.Reclamation{
		ID:            r.ID,
		CreatedAt:     r.CreatedAt,
		TicketID:      r.TicketID,
		PrestataireID: r.PrestataireID,
		Texte:         r.Texte,
		StatutID:      r.StatutID,
		DateEnvoi:     r.DateEnvoi,
		DateReponse:   r.DateReponse,
		ReponseTexte:  r.ReponseTexte,
		CreePar:       r.CreePar,
	}
}

// reclamationInsert est la charge utile d'insertion d'une Reclamation :
// uniquement les colonnes fournies par l'appelant.
type reclamationInsert struct {
	TicketID      int64      `json:"ticket_id"`
	PrestataireID int64      `json:"prestataire_id"`
	Texte         string     `json:"texte"`
	StatutID      int64      `json:"statut_id"`
	DateEnvoi     *time.Time `json:"date_envoi,omitempty"`
	CreePar       *int64     `json:"cree_par,omitempty"`
}

// InsertReclamation insère une nouvelle Reclamation (phase 5.3.1, statut
// "envoyee") et retourne la ligne créée.
func (c *Client) InsertReclamation(ctx context.Context, r *domain.Reclamation) (*domain.Reclamation, error) {
	payload := []reclamationInsert{{
		TicketID:      r.TicketID,
		PrestataireID: r.PrestataireID,
		Texte:         r.Texte,
		StatutID:      r.StatutID,
		DateEnvoi:     r.DateEnvoi,
		CreePar:       r.CreePar,
	}}
	var rows []reclamationRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/reclamation", payload, "return=representation", &rows); err != nil {
		return nil, fmt.Errorf("repository: insertion reclamation (ticket_id=%d): %w", r.TicketID, err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion reclamation: %d ligne(s) retournée(s), 1 attendue", len(rows))
	}
	return rows[0].toDomain(), nil
}

// reclamationReponsePatch est la charge utile d'enregistrement de la
// réponse du prestataire à une Reclamation.
type reclamationReponsePatch struct {
	StatutID     int64     `json:"statut_id"`
	DateReponse  time.Time `json:"date_reponse"`
	ReponseTexte string    `json:"reponse_texte"`
}

// EnregistrerReponseReclamation enregistre la réponse du prestataire
// (phase 5.3.2) : accepte (statut "acceptee") ou refuse (statut "refusee",
// litige) — statutID résolu par l'appelant (cf. domain.Reclamation
// Statut*).
func (c *Client) EnregistrerReponseReclamation(ctx context.Context, reclamationID, statutID int64, dateReponse time.Time, reponseTexte string) error {
	path := fmt.Sprintf("/reclamation?id=eq.%d", reclamationID)
	payload := reclamationReponsePatch{StatutID: statutID, DateReponse: dateReponse, ReponseTexte: reponseTexte}
	if err := c.do(ctx, http.MethodPatch, path, payload, nil); err != nil {
		return fmt.Errorf("repository: enregistrement réponse reclamation id=%d: %w", reclamationID, err)
	}
	return nil
}

// FindReclamationByID retrouve une Reclamation par son ID. Retourne (nil,
// nil), sans erreur, si aucune réclamation ne correspond.
func (c *Client) FindReclamationByID(ctx context.Context, id int64) (*domain.Reclamation, error) {
	var rows []reclamationRow
	path := fmt.Sprintf("/reclamation?select=*&id=eq.%d&limit=1", id)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche reclamation id=%d: %w", id, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}

// ListReclamationsByTicket retourne toutes les Reclamation d'un Ticket
// (plusieurs cycles réclamation/nouvelle intervention sont possibles).
func (c *Client) ListReclamationsByTicket(ctx context.Context, ticketID int64) ([]*domain.Reclamation, error) {
	var rows []reclamationRow
	path := fmt.Sprintf("/reclamation?select=*&ticket_id=eq.%d", ticketID)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: listage reclamation ticket_id=%d: %w", ticketID, err)
	}
	result := make([]*domain.Reclamation, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.toDomain())
	}
	return result, nil
}
