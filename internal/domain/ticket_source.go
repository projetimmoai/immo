package domain

import "time"

// Valeurs connues de ticket_source_type.description — le canal par lequel
// un Ticket a été signalé : un e-mail, un message envoyé depuis une future
// application, un appel téléphonique noté par un collaborateur, ou une
// saisie manuelle directe (un gestionnaire crée le ticket lui-même, sans
// canal externe — TicketSource existe quand même, avec ce type, pour que
// tout Ticket ait toujours une source identifiable de façon uniforme).
const (
	TicketSourceTypeEmail               = "email"
	TicketSourceTypeMessageApplication  = "message_application"
	TicketSourceTypeMessageTelephonique = "message_telephonique"
	TicketSourceTypeManuel              = "manuel"
)

// TicketSourceType est la table de référence des canaux d'origine d'un
// Ticket.
type TicketSourceType struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Valeurs connues de ticket_source_statut_traitement.description — le
// pipeline de traitement d'une TicketSource (identification de
// l'expéditeur, de la copropriete, de l'action...), pas le statut du/des
// Ticket qui en résultent (cf. domain.TicketStatut, une notion différente
// et indépendante).
const (
	TicketSourceStatutNouveau   = "nouveau"
	TicketSourceStatutClassifie = "classifie"
	TicketSourceStatutTraite    = "traite"
	TicketSourceStatutIgnore    = "ignore"
	TicketSourceStatutErreur    = "erreur"
)

// TicketSourceStatutTraitement est une table de référence pour le statut de
// traitement d'une TicketSource par le worker (nouveau, classifie, traite...).
type TicketSourceStatutTraitement struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// TicketSource rassemble ce qui est commun à tout canal d'origine d'un
// Ticket, quel qu'il soit : qui l'a émis (si identifiable), pour quelle
// copropriete/lot (si déterminable), et où en est son traitement. Les
// champs propres à un canal précis restent sur sa propre table détail
// (Email, MessageApplication, MessageTelephonique — reliées par TicketSourceID,
// 1-1 ; "manuel" n'a pas de table détail, rien à y stocker de plus) — même
// principe que Ticket/Incident/Sinistre.
//
// Un Ticket pointe toujours vers exactement une TicketSource (Ticket.SourceID,
// NOT NULL) — y compris une saisie manuelle, via le type "manuel".
type TicketSource struct {
	ID                 int64
	CreatedAt          time.Time
	TypeID             int64 // FK -> ticket_source_type.id, NOT NULL
	DateReception      time.Time
	PersonneID         *int64 // FK -> personne.id, résolu si l'émetteur est connu
	CoproprieteID      *int64 // FK -> copropriete.id, résolu si identifiable
	LotID              *int64 // FK -> lot.id, résolu si identifiable
	StatutTraitementID int64  // FK -> ticket_source_statut_traitement.id, NOT NULL : à fixer explicitement à l'insertion
	TraiteLe           *time.Time
	TraitePar          *int64 // FK -> personne.id (collaborateur)
	ErreurTraitement   *string
}

// MessageApplication complète une TicketSource (cf. TicketSource) avec le
// contenu d'un message envoyé depuis l'application (pas encore construite).
type MessageApplication struct {
	TicketSourceID int64 // PK -> ticket_source.id (1-1, pas d'id propre)
	Contenu        string
}

// MessageTelephonique complète une TicketSource (cf. TicketSource) avec le
// contenu noté par un collaborateur suite à un appel téléphonique.
type MessageTelephonique struct {
	TicketSourceID int64 // PK -> ticket_source.id (1-1, pas d'id propre)
	Contenu        string
}
