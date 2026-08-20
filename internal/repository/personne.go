package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// personneRow est la représentation JSON d'une ligne de la table personne.
type personneRow struct {
	ID                int64     `json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	Tel               *string   `json:"tel"`
	AdresseLigne1     *string   `json:"adresse_ligne1"`
	AdresseLigne2     *string   `json:"adresse_ligne2"`
	AdresseVille      *string   `json:"adresse_ville"`
	AdresseRegion     *string   `json:"adresse_region"`
	AdresseCodePostal *string   `json:"adresse_code_postal"`
	AdressePaysCode   *string   `json:"adresse_pays_code"`
	Email             *string   `json:"email"`
	EstPhysique       *bool     `json:"est_physique"`
	Reference         string    `json:"reference"`
	IBAN              *string   `json:"iban"`
	BIC               *string   `json:"bic"`
	EstCollaborateur  *bool     `json:"est_collaborateur"`
	CreePar           *int64    `json:"cree_par"`
}

func (r personneRow) toDomain() *domain.Personne {
	return &domain.Personne{
		ID:                r.ID,
		CreatedAt:         r.CreatedAt,
		Tel:               r.Tel,
		AdresseLigne1:     r.AdresseLigne1,
		AdresseLigne2:     r.AdresseLigne2,
		AdresseVille:      r.AdresseVille,
		AdresseRegion:     r.AdresseRegion,
		AdresseCodePostal: r.AdresseCodePostal,
		AdressePaysCode:   r.AdressePaysCode,
		Email:             r.Email,
		EstPhysique:       r.EstPhysique,
		Reference:         r.Reference,
		IBAN:              r.IBAN,
		BIC:               r.BIC,
		EstCollaborateur:  r.EstCollaborateur,
		CreePar:           r.CreePar,
	}
}

// FindPersonneByEmail cherche une Personne par adresse e-mail exacte.
// Retourne (nil, nil), sans erreur, si aucune Personne ne correspond — ce
// n'est pas une situation anormale (expéditeur inconnu du système).
func (c *Client) FindPersonneByEmail(ctx context.Context, email string) (*domain.Personne, error) {
	var rows []personneRow
	path := "/personne?select=*&email=eq." + escapeFilterValue(email) + "&limit=1"
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche personne par email (%s): %w", email, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}

// GetPersonne récupère une Personne par son ID. Retourne (nil, nil), sans
// erreur, si l'ID n'existe pas.
func (c *Client) GetPersonne(ctx context.Context, id int64) (*domain.Personne, error) {
	var rows []personneRow
	path := fmt.Sprintf("/personne?select=*&id=eq.%d&limit=1", id)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: récupération personne id=%d: %w", id, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}
