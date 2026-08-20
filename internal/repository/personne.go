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
	EstGestionnaire   *bool     `json:"est_gestionnaire"`
	EstOccupant       *bool     `json:"est_occupant"`
	EstClient         *bool     `json:"est_client"`
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
		EstGestionnaire:   r.EstGestionnaire,
		EstOccupant:       r.EstOccupant,
		EstClient:         r.EstClient,
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

// personnePhysiqueRow est la représentation JSON d'une ligne de la table personne_physique.
type personnePhysiqueRow struct {
	ID         int64     `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	Nom        *string   `json:"nom"`
	Prenom     *string   `json:"prenom"`
	PersonneID *int64    `json:"personne_id"`
	GenreID    *int64    `json:"genre_id"`
}

func (r personnePhysiqueRow) toDomain() *domain.PersonnePhysique {
	return &domain.PersonnePhysique{
		ID:         r.ID,
		CreatedAt:  r.CreatedAt,
		Nom:        r.Nom,
		Prenom:     r.Prenom,
		PersonneID: r.PersonneID,
		GenreID:    r.GenreID,
	}
}

// FindPersonnePhysiqueByPersonneID récupère la ligne PersonnePhysique
// associée à une Personne (pertinent quand Personne.EstPhysique est vrai).
// Retourne (nil, nil), sans erreur, si aucune ligne ne correspond.
func (c *Client) FindPersonnePhysiqueByPersonneID(ctx context.Context, personneID int64) (*domain.PersonnePhysique, error) {
	var rows []personnePhysiqueRow
	path := fmt.Sprintf("/personne_physique?select=*&personne_id=eq.%d&limit=1", personneID)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche personne_physique par personne_id=%d: %w", personneID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}

// personneMoraleRow est la représentation JSON d'une ligne de la table personne_morale.
type personneMoraleRow struct {
	ID                int64     `json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	Nom               *string   `json:"nom"`
	EmailFactures     *string   `json:"email_factures"`
	EstCabinetGestion *bool     `json:"est_cabinet_gestion"`
	FormeJuridiqueID  *int64    `json:"forme_juridique_id"`
	PersonneID        *int64    `json:"personne_id"`
	EstFournisseur    *bool     `json:"est_fournisseur"`
}

func (r personneMoraleRow) toDomain() *domain.PersonneMorale {
	return &domain.PersonneMorale{
		ID:                r.ID,
		CreatedAt:         r.CreatedAt,
		Nom:               r.Nom,
		EmailFactures:     r.EmailFactures,
		EstCabinetGestion: r.EstCabinetGestion,
		FormeJuridiqueID:  r.FormeJuridiqueID,
		PersonneID:        r.PersonneID,
		EstFournisseur:    r.EstFournisseur,
	}
}

// FindPersonneMoraleByPersonneID récupère la ligne PersonneMorale associée à
// une Personne (pertinent quand Personne.EstPhysique est faux). Retourne
// (nil, nil), sans erreur, si aucune ligne ne correspond.
func (c *Client) FindPersonneMoraleByPersonneID(ctx context.Context, personneID int64) (*domain.PersonneMorale, error) {
	var rows []personneMoraleRow
	path := fmt.Sprintf("/personne_morale?select=*&personne_id=eq.%d&limit=1", personneID)
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche personne_morale par personne_id=%d: %w", personneID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}
