package domain

import "time"

// Valeurs connues de ticket_statut.description — vocabulaire de cycle de
// vie partagé par tous les types de ticket (incident, sinistre, travaux...).
// "en_attente_*" indique qui doit agir ensuite : l'émetteur (client), le
// cabinet, ou un tiers (prestataire...).
const (
	TicketStatutNouveau                    = "nouveau"
	TicketStatutEnCours                    = "en_cours"
	TicketStatutEnAttenteEmetteur          = "en_attente_emetteur"
	TicketStatutEnAttenteGestionnaire      = "en_attente_gestionnaire"
	TicketStatutEnAttenteTiers             = "en_attente_tiers"
	TicketStatutEnAttenteConseilSyndical   = "en_attente_conseil_syndical"
	TicketStatutEnAttenteAssembleeGenerale = "en_attente_assemblee_generale"
	TicketStatutResolu                     = "resolu"
	TicketStatutFerme                      = "ferme"

	// TicketStatutLitige : réclamation refusée par le prestataire (phase
	// 5.3.5) — paiement suspendu (déjà garanti par la vérification négative
	// sous-jacente, cf. Incident.VerificationResultatID et service.
	// IncidentService.MettreEnPaiement), résolution hors du périmètre de ce
	// graphe (traitée séparément).
	TicketStatutLitige = "litige"
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
// ActionID vit ici, sur Ticket, et pas sur TicketSource : un même e-mail
// (ou autre source) peut donner lieu à plusieurs demandes distinctes (cf.
// email.routerVersActions), donc à plusieurs Ticket — TicketSource ne porte
// que ce qui est déterminé une seule fois par source (CoproprieteID/LotID),
// pas l'action.
//
// SourceID est NOT NULL : tout Ticket a une TicketSource, y compris une
// saisie manuelle (type "manuel", cf. domain.TicketSourceType) — pas de
// champ Declarant séparé, la personne à l'origine se lit sur
// TicketSource.PersonneID (via SourceID).
//
// ParentID permet une arborescence de tickets : un ticket créé en réaction
// à un autre (ex: un gestionnaire ouvre un ticket vers un prestataire à la
// suite d'un incident signalé par un occupant) référence son ticket
// d'origine — nul pour un ticket racine. Même principe que
// SousAction.ParentID (auto-référence, profondeur arbitraire).
type Ticket struct {
	ID              int64
	CreatedAt       time.Time
	Reference       string // référence lisible (ex: "TIC1"), générée par la base (DEFAULT), jamais fournie à l'insertion
	ActionID        int64  // FK -> action.id, NOT NULL
	SousActionID    *int64 // FK -> sous_action.id
	StatutID        int64  // FK -> ticket_statut.id, NOT NULL : à fixer explicitement à l'insertion
	SourceID        int64  // FK -> ticket_source.id, NOT NULL
	CoproprieteID   int64  // FK -> copropriete.id, NOT NULL
	LotID           *int64 // FK -> lot.id, nul = partie commune
	ParentID        *int64 // FK -> ticket.id, nul si ticket racine
	AssigneA        *int64 // FK -> personne.id (collaborateur en charge)
	CreePar         *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
	DateDeclaration time.Time
}
