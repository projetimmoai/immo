package repository

import (
	"context"
	"fmt"
	"net/http"
)

// referenceRow correspond à la forme commune de toutes les tables de
// référence du schéma (id, created_at, description).
type referenceRow struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
}

// lookupReferenceID retrouve l'ID d'une ligne de table de référence à partir
// de sa description exacte (ex: "nouveau" dans email_statut_traitement).
// Les tables de référence sont éditables librement en base : on ne code
// jamais un ID en dur, toujours une recherche par description.
func (c *Client) lookupReferenceID(ctx context.Context, table, description string) (int64, error) {
	var rows []referenceRow
	path := fmt.Sprintf("/%s?select=id,description&description=eq.%s&limit=1", table, escapeFilterValue(description))
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return 0, fmt.Errorf("repository: recherche %s.description=%q: %w", table, description, err)
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("repository: aucune ligne dans %s avec description=%q", table, description)
	}
	return rows[0].ID, nil
}

// ActionID retrouve l'ID de action pour la description donnée (voir les
// constantes domain.Action*).
func (c *Client) ActionID(ctx context.Context, description string) (int64, error) {
	return c.lookupReferenceID(ctx, "action", description)
}

// SousActionID retrouve l'ID de sous_action pour une Action et une
// description données. Une sous_action est scoped à son Action parente (sa
// description n'est unique que parmi les sous_action de cette Action), d'où
// une recherche dédiée plutôt que lookupReferenceID (qui ne filtre que sur
// la description).
func (c *Client) SousActionID(ctx context.Context, actionID int64, description string) (int64, error) {
	var rows []referenceRow
	path := fmt.Sprintf("/sous_action?select=id,description&action_id=eq.%d&description=eq.%s&limit=1", actionID, escapeFilterValue(description))
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return 0, fmt.Errorf("repository: recherche sous_action.description=%q pour action_id=%d: %w", description, actionID, err)
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("repository: aucune ligne dans sous_action avec action_id=%d et description=%q", actionID, description)
	}
	return rows[0].ID, nil
}

// EmailStatutTraitementID retrouve l'ID de email_statut_traitement pour la
// description donnée (voir les constantes domain.EmailStatut*).
func (c *Client) EmailStatutTraitementID(ctx context.Context, description string) (int64, error) {
	return c.lookupReferenceID(ctx, "email_statut_traitement", description)
}

// IncidentStatutID retrouve l'ID de incident_statut pour la description
// donnée (voir les constantes domain.IncidentStatut*).
func (c *Client) IncidentStatutID(ctx context.Context, description string) (int64, error) {
	return c.lookupReferenceID(ctx, "incident_statut", description)
}

// NiveauUrgenceID retrouve l'ID de niveau_urgence pour la description donnée
// (voir les constantes domain.NiveauUrgence*).
func (c *Client) NiveauUrgenceID(ctx context.Context, description string) (int64, error) {
	return c.lookupReferenceID(ctx, "niveau_urgence", description)
}

// SinistreStatutID retrouve l'ID de sinistre_statut pour la description
// donnée (voir les constantes domain.SinistreStatut*).
func (c *Client) SinistreStatutID(ctx context.Context, description string) (int64, error) {
	return c.lookupReferenceID(ctx, "sinistre_statut", description)
}
