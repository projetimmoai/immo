package repository

import (
	"context"
	"fmt"
	"net/http"
)

// ContratAssocie décrit un contrat où la personne recherchée est
// l'entreprise prestataire, avec la copropriété correspondante — résultat
// d'une seule requête PostgREST à ressource imbriquée (contrat ->
// copropriete), plutôt que plusieurs appels séparés.
type ContratAssocie struct {
	ContratID            int64
	NumeroContrat        *string
	CategorieTechniqueID *int64
	DateDebut            *Date // colonne SQL "date" (pas "timestamptz"), cf. Date
	DateFin              *Date
	CoproprieteID        int64
	CoproprieteNom       *string
	CoproprieteReference string
}

// contratAssocieRow reflète la forme de la réponse JSON avec ressource
// imbriquée de ListContratsParPrestataire.
type contratAssocieRow struct {
	ID                   int64   `json:"id"`
	NumeroContrat        *string `json:"numero_contrat"`
	CategorieTechniqueID *int64  `json:"categorie_technique_id"`
	DateDebut            *Date   `json:"date_debut"`
	DateFin              *Date   `json:"date_fin"`
	Copropriete          *struct {
		ID        int64   `json:"id"`
		Nom       *string `json:"nom"`
		Reference string  `json:"reference"`
	} `json:"copropriete"`
}

// ListContratsParPrestataire retourne, en un seul appel REST, tous les
// contrats où la Personne donnée est l'entreprise prestataire
// (contrat.entreprise_id), avec la copropriété correspondante — utilisé
// notamment pour enrichir le contexte d'un e-mail (internal/email) quand
// son expéditeur est une personne_morale prestataire.
func (c *Client) ListContratsParPrestataire(ctx context.Context, entrepriseID int64) ([]ContratAssocie, error) {
	path := fmt.Sprintf(
		"/contrat?select=id,numero_contrat,categorie_technique_id,date_debut,date_fin,copropriete(id,nom,reference)&entreprise_id=eq.%d",
		entrepriseID,
	)
	var rows []contratAssocieRow
	if err := c.do(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, fmt.Errorf("repository: recherche des contrats du prestataire personne id=%d: %w", entrepriseID, err)
	}

	result := make([]ContratAssocie, 0, len(rows))
	for _, r := range rows {
		ca := ContratAssocie{
			ContratID:            r.ID,
			NumeroContrat:        r.NumeroContrat,
			CategorieTechniqueID: r.CategorieTechniqueID,
			DateDebut:            r.DateDebut,
			DateFin:              r.DateFin,
		}
		if r.Copropriete != nil {
			ca.CoproprieteID = r.Copropriete.ID
			ca.CoproprieteNom = r.Copropriete.Nom
			ca.CoproprieteReference = r.Copropriete.Reference
		}
		result = append(result, ca)
	}
	return result, nil
}
