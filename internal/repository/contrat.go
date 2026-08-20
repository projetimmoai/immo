package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// contratRow est la représentation JSON d'une ligne de la table contrat.
type contratRow struct {
	ID                 int64      `json:"id"`
	CreatedAt          time.Time  `json:"created_at"`
	CoproprieteID      *int64     `json:"copropriete_id"`
	ContratTypeID      *int64     `json:"contrat_type_id"`
	EntrepriseID       *int64     `json:"entreprise_id"`
	NumeroContrat      *string    `json:"numero_contrat"`
	DateDebut          *time.Time `json:"date_debut"`
	DateFin            *time.Time `json:"date_fin"`
	DureeMois          *int64     `json:"duree_mois"`
	TaciteReconduction *bool      `json:"tacite_reconduction"`
	PreavisJours       *int64     `json:"preavis_jours"`
	CreePar            *int64     `json:"cree_par"`
}

func (r contratRow) toDomain() *domain.Contrat {
	return &domain.Contrat{
		ID:                 r.ID,
		CreatedAt:          r.CreatedAt,
		CoproprieteID:      r.CoproprieteID,
		ContratTypeID:      r.ContratTypeID,
		EntrepriseID:       r.EntrepriseID,
		NumeroContrat:      r.NumeroContrat,
		DateDebut:          r.DateDebut,
		DateFin:            r.DateFin,
		DureeMois:          r.DureeMois,
		TaciteReconduction: r.TaciteReconduction,
		PreavisJours:       r.PreavisJours,
		CreePar:            r.CreePar,
	}
}

// ListContratsParFournisseur retourne tous les Contrat où la Personne donnée
// est l'entreprise fournisseur (contrat.entreprise_id) — utilisé notamment
// pour enrichir le contexte d'un e-mail (internal/email) quand son
// expéditeur est une personne_morale fournisseur.
func (c *Client) ListContratsParFournisseur(ctx context.Context, entrepriseID int64) ([]*domain.Contrat, error) {
	var rows []contratRow
	path := fmt.Sprintf("/contrat?select=*&entreprise_id=eq.%d", entrepriseID)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche des contrats du fournisseur personne id=%d: %w", entrepriseID, err)
	}
	result := make([]*domain.Contrat, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.toDomain())
	}
	return result, nil
}
