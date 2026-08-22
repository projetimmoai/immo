package domain

import "time"

// Valeurs connues de niveau_urgence.description.
const (
	NiveauUrgenceFaible   = "faible"
	NiveauUrgenceMoyen    = "moyen"
	NiveauUrgenceEleve    = "eleve"
	NiveauUrgenceCritique = "critique"
)

// CategorieTechnique est un catalogue de domaines techniques (fuite, panne
// d'ascenseur, ménage, VMC...), partagé entre Incident (type de panne
// signalée) et Contrat (domaine couvert par le contrat).
type CategorieTechnique struct {
	ID          int64
	CreatedAt   time.Time
	Description *string
}

// NiveauUrgence est une table de référence pour le niveau d'urgence d'un Incident.
type NiveauUrgence struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Incident complète un Ticket (cf. domain.Ticket, qui porte le squelette
// commun : statut, déclarant, copropriete/lot, e-mail d'origine...) avec les
// champs propres à un dysfonctionnement technique ponctuel — par opposition
// à un sinistre (dégât des eaux, incendie...) ou des travaux déjà planifiés.
type Incident struct {
	TicketID             int64  // PK -> ticket.id (1-1, pas d'id propre)
	CategorieTechniqueID *int64 // FK -> categorie_technique.id
	UrgenceID            *int64 // FK -> niveau_urgence.id
	DateResolution       *time.Time
}
