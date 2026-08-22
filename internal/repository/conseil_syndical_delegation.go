package repository

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// conseilSyndicalDelegationRow est la représentation JSON d'une ligne de la
// table conseil_syndical_delegation.
type conseilSyndicalDelegationRow struct {
	ID              int64     `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	CoproprieteID   int64     `json:"copropriete_id"`
	PlafondCentimes int64     `json:"plafond_centimes"`
	DateDebut       Date      `json:"date_debut"`
	DateFin         Date      `json:"date_fin"`
	DateVoteAG      *Date     `json:"date_vote_ag"`
	CreePar         *int64    `json:"cree_par"`
}

func (r conseilSyndicalDelegationRow) toDomain() *domain.ConseilSyndicalDelegation {
	return &domain.ConseilSyndicalDelegation{
		ID:              r.ID,
		CreatedAt:       r.CreatedAt,
		CoproprieteID:   r.CoproprieteID,
		PlafondCentimes: r.PlafondCentimes,
		DateDebut:       r.DateDebut.Time,
		DateFin:         r.DateFin.Time,
		DateVoteAG:      dateToTimePtr(r.DateVoteAG),
		CreePar:         r.CreePar,
	}
}

// conseilSyndicalDelegationInsert est la charge utile d'insertion d'une
// ConseilSyndicalDelegation : uniquement les colonnes fournies par
// l'appelant.
type conseilSyndicalDelegationInsert struct {
	CoproprieteID   int64  `json:"copropriete_id"`
	PlafondCentimes int64  `json:"plafond_centimes"`
	DateDebut       Date   `json:"date_debut"`
	DateFin         Date   `json:"date_fin"`
	DateVoteAG      *Date  `json:"date_vote_ag,omitempty"`
	CreePar         *int64 `json:"cree_par,omitempty"`
}

// InsertConseilSyndicalDelegation insère une nouvelle délégation de pouvoir
// au conseil syndical (enveloppe C, art. 21-1 à 21-5) et retourne la ligne
// créée. Votée en assemblée générale — pas encore d'écran/API dédié pour la
// saisir ; cette fonction existe pour que le reste du cycle de vie
// (FindDelegationActive) ait quelque chose à interroger dès qu'une
// délégation est enregistrée, par quelque moyen que ce soit.
func (c *Client) InsertConseilSyndicalDelegation(ctx context.Context, d *domain.ConseilSyndicalDelegation) (*domain.ConseilSyndicalDelegation, error) {
	payload := []conseilSyndicalDelegationInsert{{
		CoproprieteID:   d.CoproprieteID,
		PlafondCentimes: d.PlafondCentimes,
		DateDebut:       Date{Time: d.DateDebut},
		DateFin:         Date{Time: d.DateFin},
		DateVoteAG:      timePtrToDate(d.DateVoteAG),
		CreePar:         d.CreePar,
	}}
	var rows []conseilSyndicalDelegationRow
	if err := c.doWithPrefer(ctx, http.MethodPost, "/conseil_syndical_delegation", payload, "return=representation", &rows); err != nil {
		return nil, fmt.Errorf("repository: insertion conseil_syndical_delegation (copropriete_id=%d): %w", d.CoproprieteID, err)
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("repository: insertion conseil_syndical_delegation: %d ligne(s) retournée(s), 1 attendue", len(rows))
	}
	return rows[0].toDomain(), nil
}

// FindDelegationActive cherche une délégation de pouvoir au conseil
// syndical active aujourd'hui pour la copropriété donnée, dont le plafond
// couvre montantCentimes (phase 3.4.7 du graphe). Retourne (nil, nil), sans
// erreur, si aucune délégation active ne couvre ce montant.
func (c *Client) FindDelegationActive(ctx context.Context, coproprieteID, montantCentimes int64) (*domain.ConseilSyndicalDelegation, error) {
	today := time.Now().UTC().Format(dateLayout)
	path := fmt.Sprintf(
		"/conseil_syndical_delegation?select=*&copropriete_id=eq.%d&date_debut=lte.%s&date_fin=gte.%s&plafond_centimes=gte.%d&limit=1",
		coproprieteID, today, today, montantCentimes,
	)
	var rows []conseilSyndicalDelegationRow
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche délégation CS active (copropriete_id=%d): %w", coproprieteID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0].toDomain(), nil
}
