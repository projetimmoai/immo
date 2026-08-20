// Package service contient la logique métier qui orchestre plusieurs couches
// (repository, drive...) pour réaliser une action de gestion complète — par
// exemple, créer une copropriété implique à la fois une ligne en base et une
// arborescence de dossiers Drive.
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/projetimmoai/immo/internal/domain"
)

// CreateCoproprieteInput regroupe les champs fournis par l'appelant pour
// créer une Copropriete. Nom et CreeParID sont obligatoires ; le reste est
// optionnel et peut être complété plus tard (mise à jour ultérieure).
type CreateCoproprieteInput struct {
	Nom       string // requis
	CreeParID int64  // requis : FK -> personne.id, le gestionnaire à l'origine de la création

	CabinetGestionID                  *int64
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
	CodeICS                           *string
	IBAN                              *string
	BIC                               *string
}

// coproprieteRepo est la portion de repository.Client utilisée ici — une
// interface étroite plutôt que le type concret, pour pouvoir tester la
// logique de compensation avec des faux (cf. copropriete_test.go).
type coproprieteRepo interface {
	InsertCopropriete(ctx context.Context, cop *domain.Copropriete) (*domain.Copropriete, error)
	DeleteCopropriete(ctx context.Context, id int64) error
}

// coproprieteDrive est la portion de drive.Client utilisée ici.
type coproprieteDrive interface {
	CreateCoproprieteFolders(ctx context.Context, reference, nom string) (string, error)
}

// CoproprieteService orchestre la création d'une Copropriete : insertion en
// base (qui génère l'id et la référence lisible, ex "COP3"), puis création de
// l'arborescence de dossiers Drive correspondante.
type CoproprieteService struct {
	Repo  coproprieteRepo
	Drive coproprieteDrive
}

// rollbackTimeout borne la tentative d'annulation de la ligne DB en cas
// d'échec Drive : on veut qu'elle s'exécute même si le ctx de l'appelant a
// été annulé (c'est justement ce qui peut avoir causé l'échec Drive), donc
// on ne réutilise pas ce ctx pour le rollback — mais on ne veut pas non plus
// attendre indéfiniment si Supabase est injoignable.
const rollbackTimeout = 10 * time.Second

// CreateCopropriete crée la Copropriete en base puis son arborescence Drive.
//
// Les deux systèmes (Postgres via l'API REST, Google Drive) sont distincts :
// aucune transaction ne peut les couvrir tous les deux. Pour éviter qu'une
// copropriété existe en base sans son arborescence Drive (ou l'inverse), la
// fonction échoue intégralement dans tous les cas :
//   - si l'insertion en base échoue, rien n'est créé ;
//   - si l'insertion en base réussit mais la création Drive échoue, la ligne
//     en base est automatiquement annulée (rollback) et la fonction retourne
//     une erreur comme si rien n'avait eu lieu ;
//   - si en plus ce rollback échoue (ex : panne réseau), il reste une ligne
//     orpheline en base sans dossiers Drive — cas limite qu'aucune
//     compensation ne peut éliminer (pas de transaction distribuée
//     possible). L'erreur retournée le signale explicitement, avec l'id à
//     corriger manuellement, plutôt que de le cacher.
//
// Dans tous les cas d'erreur, le message précise quelle étape a échoué
// (base de données ou Drive) et l'erreur d'origine.
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
		CabinetGestionID:                  in.CabinetGestionID,
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
		CodeICS:                           in.CodeICS,
		IBAN:                              in.IBAN,
		BIC:                               in.BIC,
	}

	created, err := s.Repo.InsertCopropriete(ctx, cop)
	if err != nil {
		return nil, fmt.Errorf("service: création copropriete: échec de l'insertion en base de données (nom=%s): %w", in.Nom, err)
	}

	if _, driveErr := s.Drive.CreateCoproprieteFolders(ctx, created.Reference, in.Nom); driveErr != nil {
		// Rollback avec un contexte propre : on veut qu'il s'exécute même si
		// ctx a expiré/été annulé (cause possible de l'échec Drive ci-dessus).
		rollbackCtx, cancel := context.WithTimeout(context.Background(), rollbackTimeout)
		defer cancel()

		if delErr := s.Repo.DeleteCopropriete(rollbackCtx, created.ID); delErr != nil {
			log.Printf(
				"service: copropriete %s (id=%d) créée en base mais échec Drive (%v) ET échec du rollback DB (%v) : intervention manuelle requise pour supprimer copropriete id=%d",
				created.Reference, created.ID, driveErr, delErr, created.ID,
			)
			return created, fmt.Errorf(
				"service: création copropriete %s: la création Drive a échoué (%w) ET l'annulation de la ligne en base a échoué aussi (%w) — la copropriete id=%d existe en base SANS dossiers Drive, à corriger manuellement",
				created.Reference, driveErr, delErr, created.ID,
			)
		}

		log.Printf("service: copropriete %s (id=%d) : échec Drive, ligne en base annulée (rollback réussi) : %v", created.Reference, created.ID, driveErr)
		return nil, fmt.Errorf("service: création copropriete: échec de la création des dossiers Drive (nom=%s, reference=%s) : %w — la ligne en base a été annulée, rien n'a été créé", in.Nom, created.Reference, driveErr)
	}

	return created, nil
}
