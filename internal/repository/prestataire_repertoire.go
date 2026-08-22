package repository

import (
	"context"
	"fmt"
	"net/http"
	"sort"
)

// prestataireCategorieTechniqueInsert est la charge utile d'insertion d'une
// PrestataireCategorieTechnique.
type prestataireCategorieTechniqueInsert struct {
	PersonneID           int64 `json:"personne_id"`
	CategorieTechniqueID int64 `json:"categorie_technique_id"`
}

// InsertPrestataireCategorieTechnique déclare qu'un prestataire (personne
// morale) sait traiter une catégorie technique donnée — indépendamment de
// tout contrat particulier (répertoire, cf. docs/cycle-vie-incident.md,
// phase 3.3.2).
func (c *Client) InsertPrestataireCategorieTechnique(ctx context.Context, personneID, categorieTechniqueID int64) error {
	payload := []prestataireCategorieTechniqueInsert{{PersonneID: personneID, CategorieTechniqueID: categorieTechniqueID}}
	if err := c.do(ctx, http.MethodPost, "/prestataire_categorie_technique", payload, nil); err != nil {
		return fmt.Errorf("repository: insertion prestataire_categorie_technique (personne_id=%d, categorie_technique_id=%d): %w", personneID, categorieTechniqueID, err)
	}
	return nil
}

// prestataireZoneInterventionInsert est la charge utile d'insertion d'une
// PrestataireZoneIntervention.
type prestataireZoneInterventionInsert struct {
	PersonneID int64  `json:"personne_id"`
	Ville      string `json:"ville"`
}

// InsertPrestataireZoneIntervention déclare qu'un prestataire (personne
// morale) couvre une ville donnée — indépendamment de tout contrat
// particulier (répertoire, cf. docs/cycle-vie-incident.md, phase 3.3.2).
func (c *Client) InsertPrestataireZoneIntervention(ctx context.Context, personneID int64, ville string) error {
	payload := []prestataireZoneInterventionInsert{{PersonneID: personneID, Ville: ville}}
	if err := c.do(ctx, http.MethodPost, "/prestataire_zone_intervention", payload, nil); err != nil {
		return fmt.Errorf("repository: insertion prestataire_zone_intervention (personne_id=%d, ville=%s): %w", personneID, ville, err)
	}
	return nil
}

// personneIDRow reflète la forme {"personne_id": N} commune aux deux
// requêtes de FindPrestatairesRepertoire.
type personneIDRow struct {
	PersonneID int64 `json:"personne_id"`
}

// FindPrestatairesRepertoire cherche, dans le répertoire de prestataires
// (indépendamment de tout contrat), ceux qui couvrent à la fois la
// catégorie technique et la ville données (phase 3.3.2 du graphe de cycle
// de vie d'un incident) — l'intersection de deux requêtes plutôt qu'une
// jointure PostgREST entre deux tables de liaison distinctes. Retourne les
// ID des personnes correspondantes, triés pour un résultat déterministe ;
// un slice vide (non nil) si aucun ne correspond.
func (c *Client) FindPrestatairesRepertoire(ctx context.Context, ville string, categorieTechniqueID int64) ([]int64, error) {
	var parCategorie []personneIDRow
	pathCategorie := fmt.Sprintf("/prestataire_categorie_technique?select=personne_id&categorie_technique_id=eq.%d", categorieTechniqueID)
	if err := c.do(ctx, http.MethodGet, pathCategorie, nil, &parCategorie); err != nil {
		return nil, fmt.Errorf("repository: recherche répertoire par catégorie technique id=%d: %w", categorieTechniqueID, err)
	}
	if len(parCategorie) == 0 {
		return []int64{}, nil
	}

	var parZone []personneIDRow
	pathZone := fmt.Sprintf("/prestataire_zone_intervention?select=personne_id&ville=eq.%s", escapeFilterValue(ville))
	if err := c.do(ctx, http.MethodGet, pathZone, nil, &parZone); err != nil {
		return nil, fmt.Errorf("repository: recherche répertoire par ville=%q: %w", ville, err)
	}

	dansZone := make(map[int64]bool, len(parZone))
	for _, z := range parZone {
		dansZone[z.PersonneID] = true
	}

	dejaVu := make(map[int64]bool, len(parCategorie))
	result := make([]int64, 0, len(parCategorie))
	for _, cat := range parCategorie {
		if dansZone[cat.PersonneID] && !dejaVu[cat.PersonneID] {
			dejaVu[cat.PersonneID] = true
			result = append(result, cat.PersonneID)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}
