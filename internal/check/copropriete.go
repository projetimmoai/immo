// Package check contient les vérifications de cohérence entre la base de
// données et le Drive. La création d'une copropriété (internal/service)
// n'est pas atomique entre les deux systèmes, et sa compensation (rollback)
// peut elle-même échouer dans de rares cas (cf. doc de
// service.CoproprieteService.CreateCopropriete) — ce package sert à détecter
// après coup les désynchronisations que la compensation n'aurait pas pu
// éviter, pour intervention manuelle ou automatisée (cron, à venir).
//
// Portée actuelle : uniquement les dossiers de copropriété (dossier racine
// "<reference> - <nom>" + ses sous-dossiers de catégorie,
// drive.CoproprieteSousDossiers). À étendre au fur et à mesure que d'autres
// tables/dossiers Drive sont ajoutés au projet.
package check

import (
	"context"
	"fmt"
	"strings"

	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/drive"
)

// Catégories d'anomalie retournées par CheckCoproprietes.
const (
	// CoproprieteDossierManquant : une copropriete existe en base mais aucun
	// dossier Drive correspondant n'a été trouvé sous le dossier racine.
	CoproprieteDossierManquant = "copropriete-dossier-manquant"
	// CoproprieteDossierAmbigu : plusieurs dossiers Drive correspondent à la
	// même copropriete (ne devrait pas arriver, EnsureFolder est censé
	// l'empêcher — mais un renommage manuel dans Drive peut créer le cas).
	CoproprieteDossierAmbigu = "copropriete-dossier-ambigu"
	// CoproprieteSousDossierManquant : le dossier de la copropriete existe,
	// mais un de ses sous-dossiers de catégorie (contrats, sinistres...) manque.
	CoproprieteSousDossierManquant = "copropriete-sous-dossier-manquant"
	// DossierCoproprieteOrphelin : un dossier Drive existe sous le dossier
	// racine (hors "_cabinet") sans copropriete correspondante en base — ex :
	// une copropriete supprimée en base sans que son dossier Drive le soit.
	DossierCoproprieteOrphelin = "dossier-copropriete-orphelin"
)

// dossiersIgnores sont les dossiers sous la racine qui ne correspondent pas
// à une copropriete et ne doivent donc jamais être signalés comme orphelins.
var dossiersIgnores = map[string]bool{
	"_cabinet": true,
}

// Anomalie décrit une incohérence détectée entre la base et le Drive.
type Anomalie struct {
	Categorie     string
	CoproprieteID *int64 // nil si l'anomalie ne correspond à aucune ligne DB (ex: dossier orphelin)
	Reference     string // reference de la copropriete si connue, "" sinon
	Message       string // description lisible, avec assez de contexte pour agir
}

// coproprieteLister est la portion de repository.Client utilisée ici — une
// interface étroite pour pouvoir tester avec des faux (cf. copropriete_test.go).
type coproprieteLister interface {
	ListCopropriete(ctx context.Context) ([]*domain.Copropriete, error)
}

// folderLister est la portion de drive.Client utilisée ici.
type folderLister interface {
	RootFolderID() string
	ListChildFolders(ctx context.Context, parentID string) ([]drive.Folder, error)
}

// CheckCoproprietes compare l'ensemble des coproprietes en base à
// l'arborescence Drive sous le dossier racine, et retourne toutes les
// incohérences trouvées (liste vide, non nil, si tout est cohérent).
//
// Le rapprochement dossier <-> copropriete se fait par préfixe de nom
// "<reference> - " (pas par égalité stricte sur le nom complet) : si le nom
// d'une copropriete change en base sans que le dossier Drive soit renommé,
// ce n'est volontairement pas signalé comme une anomalie ici (aucune
// fonctionnalité de renommage synchronisé n'existe encore, cf. NOTES.md) —
// seule la reference, stable, fait foi pour le rapprochement.
func CheckCoproprietes(ctx context.Context, repo coproprieteLister, drv folderLister) ([]Anomalie, error) {
	coproprietes, err := repo.ListCopropriete(ctx)
	if err != nil {
		return nil, fmt.Errorf("check: récupération des coproprietes: %w", err)
	}
	rootFolders, err := drv.ListChildFolders(ctx, drv.RootFolderID())
	if err != nil {
		return nil, fmt.Errorf("check: récupération des dossiers Drive racine: %w", err)
	}

	anomalies := []Anomalie{}

	// matched suit les dossiers racine rapprochés d'une copropriete, pour
	// détecter les orphelins une fois toutes les coproprietes parcourues.
	matched := make(map[string]bool, len(rootFolders))

	for _, cop := range coproprietes {
		prefix := cop.Reference + " - "
		var candidats []drive.Folder
		for _, f := range rootFolders {
			if strings.HasPrefix(f.Name, prefix) {
				candidats = append(candidats, f)
			}
		}

		switch len(candidats) {
		case 0:
			anomalies = append(anomalies, Anomalie{
				Categorie:     CoproprieteDossierManquant,
				CoproprieteID: &cop.ID,
				Reference:     cop.Reference,
				Message:       fmt.Sprintf("copropriete %s (id=%d) : aucun dossier Drive %q trouvé sous le dossier racine", cop.Reference, cop.ID, prefix+"..."),
			})
			continue
		case 1:
			matched[candidats[0].ID] = true
		default:
			for _, f := range candidats {
				matched[f.ID] = true
			}
			anomalies = append(anomalies, Anomalie{
				Categorie:     CoproprieteDossierAmbigu,
				CoproprieteID: &cop.ID,
				Reference:     cop.Reference,
				Message:       fmt.Sprintf("copropriete %s (id=%d) : %d dossiers Drive correspondent (1 attendu) sous le dossier racine", cop.Reference, cop.ID, len(candidats)),
			})
			continue
		}

		coproprieteFolderID := candidats[0].ID
		subFolders, err := drv.ListChildFolders(ctx, coproprieteFolderID)
		if err != nil {
			return nil, fmt.Errorf("check: récupération des sous-dossiers de %s (id=%d): %w", cop.Reference, cop.ID, err)
		}
		present := make(map[string]bool, len(subFolders))
		for _, f := range subFolders {
			present[f.Name] = true
		}
		for _, cat := range drive.CoproprieteSousDossiers {
			if !present[cat] {
				anomalies = append(anomalies, Anomalie{
					Categorie:     CoproprieteSousDossierManquant,
					CoproprieteID: &cop.ID,
					Reference:     cop.Reference,
					Message:       fmt.Sprintf("copropriete %s (id=%d) : sous-dossier %q manquant", cop.Reference, cop.ID, cat),
				})
			}
		}
	}

	for _, f := range rootFolders {
		if dossiersIgnores[f.Name] || matched[f.ID] {
			continue
		}
		anomalies = append(anomalies, Anomalie{
			Categorie: DossierCoproprieteOrphelin,
			Message:   fmt.Sprintf("dossier Drive %q (id=%s) sous le dossier racine ne correspond à aucune copropriete en base", f.Name, f.ID),
		})
	}

	return anomalies, nil
}
