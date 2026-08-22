package domain

import "time"

// Valeurs connues de sinistre_resultat.description — résultat d'un dossier
// d'assurance, indépendant du statut générique du ticket (cf.
// domain.TicketStatut) : un sinistre peut être "resolu" (statut du ticket)
// avec un résultat "indemnise" ou "refuse".
const (
	SinistreResultatIndemnise = "indemnise"
	SinistreResultatRefuse    = "refuse"
)

// SinistreResultat est la table de référence du résultat d'un dossier
// d'assurance.
type SinistreResultat struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Sinistre complète un Ticket (cf. domain.Ticket, qui porte le squelette
// commun : statut, déclarant, copropriete/lot, e-mail d'origine...) avec les
// champs propres à une déclaration d'assurance.
//
// Les montants sont exprimés en centimes d'euro (entiers), conformément à la
// règle du projet — jamais en float.
type Sinistre struct {
	TicketID                 int64 // PK -> ticket.id (1-1, pas d'id propre)
	CompagnieAssurance       *string
	NumeroPolice             *string
	NumeroDossier            *string
	DateSurvenance           *time.Time // date
	MontantEstimeCentimes    *int32
	MontantIndemniseCentimes *int32
	ResultatID               *int64 // FK -> sinistre_resultat.id
}
