package domain

import "time"

// Valeurs connues de sinistre_statut.description.
const (
	SinistreStatutDeclare   = "declare"
	SinistreStatutEnCours   = "en_cours"
	SinistreStatutIndemnise = "indemnise"
	SinistreStatutClos      = "clos"
	SinistreStatutRefuse    = "refuse"
)

// SinistreStatut est une table de référence pour le statut d'un Sinistre.
type SinistreStatut struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Sinistre est une déclaration d'assurance, rattachée à une Copropriete et
// éventuellement à un Lot précis.
//
// Les montants sont exprimés en centimes d'euro (entiers), conformément à la
// règle du projet — jamais en float.
type Sinistre struct {
	ID                       int64
	CreatedAt                time.Time
	EmailOrigineID           *int64 // FK -> email.id, si déclenché par un e-mail
	CoproprieteID            int64  // FK -> copropriete.id, NOT NULL
	LotID                    *int64 // FK -> lot.id
	DeclarantID              *int64 // FK -> personne.id
	Description              *string
	CompagnieAssurance       *string
	NumeroPolice             *string
	NumeroDossier            *string
	DateSurvenance           *time.Time // date
	DateDeclaration          time.Time
	MontantEstimeCentimes    *int32
	MontantIndemniseCentimes *int32
	StatutID                 int64  // FK -> sinistre_statut.id, NOT NULL : à fixer explicitement à l'insertion
	AssigneA                 *int64 // FK -> personne.id (collaborateur en charge)
	CreePar                  *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
	Reference                string // référence lisible (ex: "SIN1"), générée par la base (DEFAULT), jamais fournie à l'insertion
}
