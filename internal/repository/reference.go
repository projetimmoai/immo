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

// SousActionID retrouve l'ID d'une sous_action pour une Action racine, un
// parent (nil pour une sous_action de premier niveau — directement sous
// l'Action —, sinon l'ID de sa sous_action parente) et une description.
// Une sous_action est scoped à ce contexte (action_id, parent_id) : sa
// description n'est unique que parmi les sous_action qui le partagent (cf.
// contrainte UNIQUE NULLS NOT DISTINCT en base), d'où une recherche dédiée
// plutôt que lookupReferenceID (qui ne filtre que sur la description).
func (c *Client) SousActionID(ctx context.Context, actionID int64, parentID *int64, description string) (int64, error) {
	parentFilter := "parent_id=is.null"
	if parentID != nil {
		parentFilter = fmt.Sprintf("parent_id=eq.%d", *parentID)
	}
	var rows []referenceRow
	path := fmt.Sprintf("/sous_action?select=id,description&action_id=eq.%d&%s&description=eq.%s&limit=1", actionID, parentFilter, escapeFilterValue(description))
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return 0, fmt.Errorf("repository: recherche sous_action.description=%q (action_id=%d, parent_id=%s): %w", description, actionID, formatParentID(parentID), err)
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("repository: aucune ligne dans sous_action avec action_id=%d, parent_id=%s et description=%q", actionID, formatParentID(parentID), description)
	}
	return rows[0].ID, nil
}

// formatParentID formate un parent_id de sous_action pour les messages
// d'erreur ("aucun" si nil, plutôt que l'adresse mémoire d'un *int64).
func formatParentID(parentID *int64) string {
	if parentID == nil {
		return "aucun (premier niveau)"
	}
	return fmt.Sprintf("%d", *parentID)
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
