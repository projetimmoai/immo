// Package service contient la logique métier qui orchestre plusieurs
// couches (repository, storage...) pour réaliser une action de gestion
// complète.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// CreateCoproprieteInput regroupe les champs fournis par l'appelant pour
// créer une Copropriete. Nom et CreeParID sont obligatoires ; le reste est
// optionnel et peut être complété plus tard (mise à jour ultérieure).
//
// Ne couvre volontairement pas les champs de CoproprieteBanque (iban, bic,
// code_ics) : cette table séparée existe justement pour que RLS puisse la
// verrouiller indépendamment de Copropriete (accès dirigeant/sys_admin
// uniquement), et rien ne les renseigne encore — pas de
// repository.InsertCoproprieteBanque pour l'instant, à ajouter via une
// fonction Postgres en RPC le jour où ce sera nécessaire (deux écritures
// liées, cf. CLAUDE.md), plutôt que deux appels REST séparés.
type CreateCoproprieteInput struct {
	Nom       string // requis
	CreeParID int64  // requis : FK -> personne.id, le gestionnaire à l'origine de la création

	SyndicID                          *int64
	EstSyndic                         *bool
	AdresseCodePostal                 *string
	AdresseVille                      *string
	AdresseRegion                     *string
	AdressePaysCode                   *string
	ExerciceDebut                     *time.Time
	ExerciceFin                       *time.Time
	AppelChargesFrequenceID           *int64
	AppelChargesDate                  *time.Time
	AppelChargesNumJoursAvantEcheance *int64
	ClesRepartition                   *float32
	ArreteComptableJour               *int64
	ArreteComptableMois               *int64
	NumeroImmatriculation             *string
	NumeroMandat                      *string
	MandatDateDebut                   *time.Time
	MandatDureeEnMois                 *int64
}

// coproprieteRepo est la portion de repository.Client utilisée ici — une
// interface étroite plutôt que le type concret, pour pouvoir tester avec
// des faux (cf. copropriete_test.go).
type coproprieteRepo interface {
	InsertCopropriete(ctx context.Context, cop *domain.Copropriete) (*domain.Copropriete, error)
}

// CoproprieteService orchestre la création d'une Copropriete.
//
// Avant l'usage d'internal/storage (Supabase Storage) à la place de Google
// Drive, cette étape créait aussi une arborescence de dossiers Drive, avec
// toute la complexité d'une compensation à deux phases (la ligne DB pouvait
// exister sans ses dossiers, et le rollback lui-même pouvait échouer).
// Storage ne nécessite aucune création à l'avance : un chemin d'objet
// (ex: "COP3 - Nom/contrats/...") existe implicitement dès qu'un fichier y
// est déposé — plus rien à orchestrer ici que l'insertion en base.
type CoproprieteService struct {
	Repo coproprieteRepo
}

// CreateCopropriete crée la Copropriete en base et retourne la ligne créée
// (avec son id et sa référence lisible, ex "COP3", générés par la base).
func (s *CoproprieteService) CreateCopropriete(ctx context.Context, in CreateCoproprieteInput) (*domain.Copropriete, error) {
	if in.Nom == "" {
		return nil, fmt.Errorf("service: création copropriete: le nom est obligatoire")
	}
	if in.CreeParID <= 0 {
		return nil, fmt.Errorf("service: création copropriete: cree_par (gestionnaire à l'origine de la création) est obligatoire")
	}

	nom := in.Nom
	creePar := in.CreeParID
	cop := &domain.Copropriete{
		Nom:                               &nom,
		CreePar:                           &creePar,
		SyndicID:                          in.SyndicID,
		EstSyndic:                         in.EstSyndic,
		AdresseCodePostal:                 in.AdresseCodePostal,
		AdresseVille:                      in.AdresseVille,
		AdresseRegion:                     in.AdresseRegion,
		AdressePaysCode:                   in.AdressePaysCode,
		ExerciceDebut:                     in.ExerciceDebut,
		ExerciceFin:                       in.ExerciceFin,
		AppelChargesFrequenceID:           in.AppelChargesFrequenceID,
		AppelChargesDate:                  in.AppelChargesDate,
		AppelChargesNumJoursAvantEcheance: in.AppelChargesNumJoursAvantEcheance,
		ClesRepartition:                   in.ClesRepartition,
		ArreteComptableJour:               in.ArreteComptableJour,
		ArreteComptableMois:               in.ArreteComptableMois,
		NumeroImmatriculation:             in.NumeroImmatriculation,
		NumeroMandat:                      in.NumeroMandat,
		MandatDateDebut:                   in.MandatDateDebut,
		MandatDureeEnMois:                 in.MandatDureeEnMois,
	}

	created, err := s.Repo.InsertCopropriete(ctx, cop)
	if err != nil {
		return nil, fmt.Errorf("service: création copropriete: échec de l'insertion en base de données (nom=%s): %w", in.Nom, err)
	}
	return created, nil
}
