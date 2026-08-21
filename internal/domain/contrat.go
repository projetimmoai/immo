package domain

import "time"

// Contrat est un contrat souscrit par une Copropriete auprès d'une entreprise
// (ascensoriste, assureur, syndic tiers...).
type Contrat struct {
	ID                   int64
	CreatedAt            time.Time
	CoproprieteID        *int64 // FK -> copropriete.id
	CategorieTechniqueID *int64 // FK -> categorie_technique.id (catalogue partagé avec Incident, cf. domain.CategorieTechnique)
	EntrepriseID         *int64 // FK -> personne.id (personne_morale prestataire)
	NumeroContrat        *string
	DateDebut            *time.Time // date
	DateFin              *time.Time // date
	DureeMois            *int64
	TaciteReconduction   *bool
	PreavisJours         *int64
	CreePar              *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
}

// FrequenceID est une table de référence pour les fréquences (mensuelle,
// trimestrielle, annuelle...), notamment utilisée par Copropriete.AppelChargesFrequenceID.
type FrequenceID struct {
	ID          int64
	CreatedAt   time.Time
	Description *string
}

// ExerciceComptable est un exercice comptable annuel d'une Copropriete.
type ExerciceComptable struct {
	ID                  int64
	CreatedAt           time.Time
	MisAJour            time.Time
	CoproprieteID       *int64     // FK -> copropriete.id
	DateDebut           *time.Time // date
	DateFin             *time.Time // date
	DateApprobation     *time.Time // date
	DateArrete          *time.Time // date
	StatutID            int64      // FK -> exercice_comptable_statut.id, NOT NULL : à fixer explicitement à l'insertion (pas de DEFAULT en base)
	CloturePar          *int64     // FK -> personne.id
	AgApprobationID     *int64     // référence une assemblée générale (pas de table dédiée pour l'instant)
	StatutApprobationID int64      // FK -> exercice_comptable_statut_approbation.id, NOT NULL : à fixer explicitement à l'insertion (pas de DEFAULT en base)
	CreePar             *int64     // FK -> personne.id (gestionnaire à l'origine de la création)
}

// Valeurs connues de exercice_comptable_statut.description (provisoires, à
// ajuster si elles ne correspondent pas au process métier réel).
const (
	ExerciceComptableStatutOuvert   = "ouvert"
	ExerciceComptableStatutCloture  = "cloture"
	ExerciceComptableStatutApprouve = "approuve"
)

// ExerciceComptableStatut est une table de référence pour le statut d'un ExerciceComptable.
type ExerciceComptableStatut struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Valeurs connues de exercice_comptable_statut_approbation.description :
// statut de l'approbation des comptes par l'assemblée générale.
const (
	StatutApprobationAGNonTenue    = "ag_non_tenue"
	StatutApprobationAGApprouve    = "approuve"
	StatutApprobationAGNonApprouve = "non_approuve"
)

// ExerciceComptableStatutApprobation est une table de référence pour le
// statut d'approbation des comptes d'un ExerciceComptable par l'AG.
type ExerciceComptableStatutApprobation struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}
