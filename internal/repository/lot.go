package repository

import (
	"context"
	"fmt"
	"net/http"
)

// LotAssocie décrit un lot associé à une personne (propriétaire, gestionnaire
// ou indivisaire), avec assez de contexte pour router un e-mail : le lot
// lui-même et la copropriété à laquelle il appartient (via son bâtiment).
// Résultat d'une seule requête PostgREST à ressources imbriquées
// (lot_personne_map -> lot -> batiment -> copropriete), plutôt que plusieurs
// appels séparés.
type LotAssocie struct {
	LotPersonneMapID     int64
	EstProprietaire      *bool
	EstOccupant          *bool
	EstGestionnaire      *bool
	EstIndivision        *bool
	Debut                *Date // colonne SQL "date" (pas "timestamptz"), cf. Date
	Fin                  *Date
	LotID                int64
	LotNumero            *string
	LotReference         string
	CoproprieteID        int64
	CoproprieteNom       *string
	CoproprieteReference string
}

// lotPersonneMapRow reflète la forme de la réponse JSON avec ressources
// imbriquées de ListLotsParPersonne.
type lotPersonneMapRow struct {
	ID              int64 `json:"id"`
	EstProprietaire *bool `json:"est_proprietaire"`
	EstOccupant     *bool `json:"est_occupant"`
	EstGestionnaire *bool `json:"est_gestionnaire"`
	EstIndivision   *bool `json:"est_indivision"`
	Debut           *Date `json:"debut"`
	Fin             *Date `json:"fin"`
	Lot             *struct {
		ID        int64   `json:"id"`
		Numero    *string `json:"numero"`
		Reference string  `json:"reference"`
		Batiment  *struct {
			ID          int64 `json:"id"`
			Copropriete *struct {
				ID        int64   `json:"id"`
				Nom       *string `json:"nom"`
				Reference string  `json:"reference"`
			} `json:"copropriete"`
		} `json:"batiment"`
	} `json:"lot"`
}

// ListLotsParPersonne retourne, en un seul appel REST, tous les lots
// associés à une Personne (propriétaire, gestionnaire ou indivisaire), avec
// la copropriété correspondante — utilisé notamment pour enrichir le
// contexte d'un e-mail (internal/email) à partir de son expéditeur.
func (c *Client) ListLotsParPersonne(ctx context.Context, personneID int64) ([]LotAssocie, error) {
	path := fmt.Sprintf(
		"/lot_personne_map?select=id,est_proprietaire,est_occupant,est_gestionnaire,est_indivision,debut,fin,lot(id,numero,reference,batiment(id,copropriete(id,nom,reference)))&personne_id=eq.%d",
		personneID,
	)
	var rows []lotPersonneMapRow
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche des lots associés à la personne id=%d: %w", personneID, err)
	}

	result := make([]LotAssocie, 0, len(rows))
	for _, r := range rows {
		la := LotAssocie{
			LotPersonneMapID: r.ID,
			EstProprietaire:  r.EstProprietaire,
			EstOccupant:      r.EstOccupant,
			EstGestionnaire:  r.EstGestionnaire,
			EstIndivision:    r.EstIndivision,
			Debut:            r.Debut,
			Fin:              r.Fin,
		}
		if r.Lot != nil {
			la.LotID = r.Lot.ID
			la.LotNumero = r.Lot.Numero
			la.LotReference = r.Lot.Reference
			if r.Lot.Batiment != nil && r.Lot.Batiment.Copropriete != nil {
				la.CoproprieteID = r.Lot.Batiment.Copropriete.ID
				la.CoproprieteNom = r.Lot.Batiment.Copropriete.Nom
				la.CoproprieteReference = r.Lot.Batiment.Copropriete.Reference
			}
		}
		result = append(result, la)
	}
	return result, nil
}
