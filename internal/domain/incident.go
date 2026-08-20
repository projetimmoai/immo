package domain

import "time"

// Valeurs connues de incident_statut.description.
const (
	IncidentStatutNouveau = "nouveau"
	IncidentStatutEnCours = "en_cours"
	IncidentStatutResolu  = "resolu"
	IncidentStatutAnnule  = "annule"
)

// Valeurs connues de niveau_urgence.description.
const (
	NiveauUrgenceFaible   = "faible"
	NiveauUrgenceMoyen    = "moyen"
	NiveauUrgenceEleve    = "eleve"
	NiveauUrgenceCritique = "critique"
)

// IncidentType est un catalogue de types de pannes/incidents techniques
// (fuite, panne d'ascenseur, panne d'interphone...).
type IncidentType struct {
	ID          int64
	CreatedAt   time.Time
	Description *string
	Consequence *string
	Urgence     *string // texte libre existant en base, distinct de la table de référence niveau_urgence
	Symptome    *string
}

// IncidentStatut est une table de référence pour le statut d'un Incident.
type IncidentStatut struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// NiveauUrgence est une table de référence pour le niveau d'urgence d'un Incident.
type NiveauUrgence struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Incident est une instance de panne/dysfonctionnement technique signalée,
// rattachée à une Copropriete et éventuellement à un Lot précis.
type Incident struct {
	ID              int64
	CreatedAt       time.Time
	IncidentTypeID  *int64 // FK -> incident_type.id
	EmailOrigineID  *int64 // FK -> email.id, si déclenché par un e-mail
	CoproprieteID   int64  // FK -> copropriete.id, NOT NULL
	LotID           *int64 // FK -> lot.id, nul = partie commune
	DeclarantID     *int64 // FK -> personne.id
	Description     *string
	UrgenceID       *int64 // FK -> niveau_urgence.id
	StatutID        int64  // FK -> incident_statut.id, NOT NULL : à fixer explicitement à l'insertion
	DateDeclaration time.Time
	DateResolution  *time.Time
	AssigneA        *int64 // FK -> personne.id (collaborateur en charge)
	CreePar         *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
	Reference       string // référence lisible (ex: "INC1"), générée par la base (DEFAULT), jamais fournie à l'insertion
}
