package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// conseilSyndicalMandatRow est la représentation JSON d'une ligne de la
// table conseil_syndical_mandat.
type conseilSyndicalMandatRow struct {
	ID            int64      `json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	PersonneID    *int64     `json:"personne_id"`
	CoproprieteID *int64     `json:"copropriete_id"`
	DateDebut     *time.Time `json:"date_debut"`
	DateFin       *time.Time `json:"date_fin"`
	StatutID      *int64     `json:"statut_id"`
	CreePar       *int64     `json:"cree_par"`
}

func (r conseilSyndicalMandatRow) toDomain() *domain.ConseilSyndicalMandat {
	return &domain.ConseilSyndicalMandat{
		ID:            r.ID,
		CreatedAt:     r.CreatedAt,
		PersonneID:    r.PersonneID,
		CoproprieteID: r.CoproprieteID,
		DateDebut:     r.DateDebut,
		DateFin:       r.DateFin,
		StatutID:      r.StatutID,
		CreePar:       r.CreePar,
	}
}

// conseilSyndicalMandatInsert est la charge utile d'insertion d'un
// ConseilSyndicalMandat : uniquement les colonnes fournies par l'appelant
// (id/created_at sont générés par la base).
type conseilSyndicalMandatInsert struct {
	PersonneID    *int64     `json:"personne_id,omitempty"`
	CoproprieteID *int64     `json:"copropriete_id,omitempty"`
	DateDebut     *time.Time `json:"date_debut,omitempty"`
	DateFin       *time.Time `json:"date_fin,omitempty"`
	StatutID      *int64     `json:"statut_id,omitempty"`
	CreePar       *int64     `json:"cree_par,omitempty"`
}

// InsertConseilSyndicalMandat insère un nouveau ConseilSyndicalMandat et
// retourne la ligne créée. m.StatutID doit être résolu explicitement par
// l'appelant (cf. ConseilSyndicalMandatStatutID, constantes
// domain.ConseilSyndicalMandatStatut*).
func (c *Client) InsertConseilSyndicalMandat(ctx context.Context, m *domain.ConseilSyndicalMandat) (*domain.ConseilSyndicalMandat, error) {
	payload := []conseilSyndicalMandatInsert{{
		PersonneID:    m.PersonneID,
		CoproprieteID: m.CoproprieteID,
		DateDebut:     m.DateDebut,
		DateFin:       m.DateFin,
		StatutID:      m.StatutID,
		CreePar:       m.CreePar,
	}}
	var rows []conseilSyndicalMandatRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/conseil_syndical_mandat", payload, "return=representation", &rows); err != nil {
		return nil, fmt.Errorf("repository: insertion conseil_syndical_mandat: %w", err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion conseil_syndical_mandat: %d ligne(s) retournée(s), 1 attendue", len(rows))
	}
	return rows[0].toDomain(), nil
}

// DeleteConseilSyndicalMandat supprime un ConseilSyndicalMandat par son ID
// (utilisé notamment par les tests d'intégration pour nettoyer après eux).
func (c *Client) DeleteConseilSyndicalMandat(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/conseil_syndical_mandat?id=eq.%d", id)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("repository: suppression conseil_syndical_mandat id=%d: %w", id, err)
	}
	return nil
}

// conseilSyndicalPresidentRow est la représentation JSON d'une ligne de la
// table conseil_syndical_president.
type conseilSyndicalPresidentRow struct {
	ID            int64      `json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	PersonneID    *int64     `json:"personne_id"`
	CoproprieteID *int64     `json:"copropriete_id"`
	DateDebut     *time.Time `json:"date_debut"`
	DateFin       *time.Time `json:"date_fin"`
	EluEnAG       *bool      `json:"elu_en_ag"`
	EluParCS      *bool      `json:"elu_par_cs"`
	CreePar       *int64     `json:"cree_par"`
}

func (r conseilSyndicalPresidentRow) toDomain() *domain.ConseilSyndicalPresident {
	return &domain.ConseilSyndicalPresident{
		ID:            r.ID,
		CreatedAt:     r.CreatedAt,
		PersonneID:    r.PersonneID,
		CoproprieteID: r.CoproprieteID,
		DateDebut:     r.DateDebut,
		DateFin:       r.DateFin,
		EluEnAG:       r.EluEnAG,
		EluParCS:      r.EluParCS,
		CreePar:       r.CreePar,
	}
}

// conseilSyndicalPresidentInsert est la charge utile d'insertion d'un
// ConseilSyndicalPresident : uniquement les colonnes fournies par
// l'appelant (id/created_at sont générés par la base).
type conseilSyndicalPresidentInsert struct {
	PersonneID    *int64     `json:"personne_id,omitempty"`
	CoproprieteID *int64     `json:"copropriete_id,omitempty"`
	DateDebut     *time.Time `json:"date_debut,omitempty"`
	DateFin       *time.Time `json:"date_fin,omitempty"`
	EluEnAG       *bool      `json:"elu_en_ag,omitempty"`
	EluParCS      *bool      `json:"elu_par_cs,omitempty"`
	CreePar       *int64     `json:"cree_par,omitempty"`
}

// InsertConseilSyndicalPresident insère un nouveau ConseilSyndicalPresident
// et retourne la ligne créée.
func (c *Client) InsertConseilSyndicalPresident(ctx context.Context, p *domain.ConseilSyndicalPresident) (*domain.ConseilSyndicalPresident, error) {
	payload := []conseilSyndicalPresidentInsert{{
		PersonneID:    p.PersonneID,
		CoproprieteID: p.CoproprieteID,
		DateDebut:     p.DateDebut,
		DateFin:       p.DateFin,
		EluEnAG:       p.EluEnAG,
		EluParCS:      p.EluParCS,
		CreePar:       p.CreePar,
	}}
	var rows []conseilSyndicalPresidentRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/conseil_syndical_president", payload, "return=representation", &rows); err != nil {
		return nil, fmt.Errorf("repository: insertion conseil_syndical_president: %w", err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion conseil_syndical_president: %d ligne(s) retournée(s), 1 attendue", len(rows))
	}
	return rows[0].toDomain(), nil
}

// DeleteConseilSyndicalPresident supprime un ConseilSyndicalPresident par
// son ID (utilisé notamment par les tests d'intégration pour nettoyer après
// eux).
func (c *Client) DeleteConseilSyndicalPresident(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/conseil_syndical_president?id=eq.%d", id)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("repository: suppression conseil_syndical_president id=%d: %w", id, err)
	}
	return nil
}

// conseilSyndicalMandatCoproprieteRow reflète la forme de la réponse JSON
// avec ressource imbriquée de ListCoproprietesConseilSyndicalParPersonne.
type conseilSyndicalMandatCoproprieteRow struct {
	Copropriete *struct {
		ID        int64   `json:"id"`
		Nom       *string `json:"nom"`
		Reference string  `json:"reference"`
	} `json:"copropriete"`
}

// ListCoproprietesConseilSyndicalParPersonne retourne toutes les
// copropriete pour lesquelles la Personne donnée a un mandat actif de
// membre du conseil syndical (conseil_syndical_mandat, statut "membre") —
// utilisé pour enrichir le contexte d'un e-mail (internal/email) quand son
// expéditeur est copropriétaire, afin de distinguer un copropriétaire
// ordinaire d'un membre du conseil syndical.
func (c *Client) ListCoproprietesConseilSyndicalParPersonne(ctx context.Context, personneID int64) ([]CoproprieteAssociee, error) {
	statutMembreID, err := c.ConseilSyndicalMandatStatutID(ctx, domain.ConseilSyndicalMandatStatutMembre)
	if err != nil {
		return nil, fmt.Errorf("repository: résolution du statut %q de conseil_syndical_mandat_statut: %w", domain.ConseilSyndicalMandatStatutMembre, err)
	}

	path := fmt.Sprintf(
		"/conseil_syndical_mandat?select=copropriete(id,nom,reference)&personne_id=eq.%d&statut_id=eq.%d",
		personneID, statutMembreID,
	)
	var rows []conseilSyndicalMandatCoproprieteRow
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche des mandats de conseil syndical de la personne id=%d: %w", personneID, err)
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
