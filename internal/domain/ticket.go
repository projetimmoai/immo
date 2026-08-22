package domain

import "time"

// Valeurs connues de ticket_statut.description — vocabulaire de cycle de
// vie partagé par tous les types de ticket (incident, sinistre, travaux...).
// "en_attente_*" indique qui doit agir ensuite : l'émetteur (client), le
// cabinet, ou un tiers (prestataire...).
const (
	TicketStatutNouveau               = "nouveau"
	TicketStatutEnCours               = "en_cours"
	TicketStatutEnAttenteEmetteur     = "en_attente_emetteur"
	TicketStatutEnAttenteGestionnaire = "en_attente_gestionnaire"
	TicketStatutEnAttenteTiers        = "en_attente_tiers"
	TicketStatutResolu                = "resolu"
	TicketStatutFerme                 = "ferme"
)

// TicketStatut est la table de référence du statut d'un Ticket — partagée
// par tous les types (incident, sinistre, travaux...), par opposition aux
// champs spécifiques à chaque type qui restent sur leur propre table (cf.
// Incident, Sinistre).
type TicketStatut struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Ticket rassemble le squelette commun à toute action donnant lieu à un
// suivi (incident, sinistre, travaux, demande_administrative, mutation...) :
// qui l'a déclarée, pour quelle copropriete/lot, son statut, qui la traite.
// Les champs spécifiques à un type d'action restent sur sa propre table
// (Incident, Sinistre...), reliée à Ticket par TicketID (1-1, la PK de
// cette table détail est aussi sa FK vers Ticket) — pas de colonnes
// nullable partagées par tous les types juste pour les besoins de
// quelques-uns (même choix de conception que pour domain.Role).
//
// ActionID vit ici, sur Ticket, et pas sur Email : un même e-mail peut
// donner lieu à plusieurs demandes distinctes (cf. email.routerVersActions),
// donc à plusieurs Ticket — Email ne porte que ce qui est déterminé une
// seule fois par e-mail (CoproprieteID/LotID), pas l'action.
type Ticket struct {
	ID              int64
	CreatedAt       time.Time
	Reference       string // référence lisible (ex: "TIC1"), générée par la base (DEFAULT), jamais fournie à l'insertion
	ActionID        int64  // FK -> action.id, NOT NULL
	SousActionID    *int64 // FK -> sous_action.id
	StatutID        int64  // FK -> ticket_statut.id, NOT NULL : à fixer explicitement à l'insertion
	EmailOrigineID  *int64 // FK -> email.id, si déclenché par un e-mail
	CoproprieteID   int64  // FK -> copropriete.id, NOT NULL
	LotID           *int64 // FK -> lot.id, nul = partie commune
	DeclarantID     *int64 // FK -> personne.id
	AssigneA        *int64 // FK -> personne.id (collaborateur en charge)
	CreePar         *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
	DateDeclaration time.Time
}
