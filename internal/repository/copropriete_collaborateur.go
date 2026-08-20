package repository

import (
	"context"
	"fmt"
	"net/http"
)

// CoproprieteAssociee décrit une copropriete associée à une personne dans un
// rôle donné, avec juste assez d'information pour l'identifier — utilisé
// notamment par internal/email pour enrichir le contexte d'un e-mail.
type CoproprieteAssociee struct {
	CoproprieteID        int64
	CoproprieteNom       *string
	CoproprieteReference string
}

// coproprieteCollaborateurRow reflète la forme de la réponse JSON avec
// ressource imbriquée de ListCoproprietesParGestionnaire.
type coproprieteCollaborateurRow struct {
	Copropriete *struct {
		ID        int64   `json:"id"`
		Nom       *string `json:"nom"`
		Reference string  `json:"reference"`
	} `json:"copropriete"`
}

// ListCoproprietesParGestionnaire retourne, en un seul appel REST, toutes
// les copropriete dont la Personne donnée est gestionnaire en charge (table
// copropriete_collaborateur_map) — utilisé notamment pour enrichir le
// contexte d'un e-mail (internal/email) quand son expéditeur est un
// gestionnaire du cabinet.
func (c *Client) ListCoproprietesParGestionnaire(ctx context.Context, personneID int64) ([]CoproprieteAssociee, error) {
	path := fmt.Sprintf(
		"/copropriete_collaborateur_map?select=copropriete(id,nom,reference)&personne_id=eq.%d",
		personneID,
	)
	var rows []coproprieteCollaborateurRow
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche des coproprietes gérées par la personne id=%d: %w", personneID, err)
	}
	result := make([]CoproprieteAssociee, 0, len(rows))
	for _, r := range rows {
		if r.Copropriete == nil {
			continue
		}
		result = append(result, CoproprieteAssociee{
			CoproprieteID:        r.Copropriete.ID,
			CoproprieteNom:       r.Copropriete.Nom,
			CoproprieteReference: r.Copropriete.Reference,
		})
	}
	return result, nil
}
