package domain

import "time"

// Valeurs connues de devis_statut.description — cycle de vie d'un Devis
// (cf. docs/cycle-vie-incident.md, phase 3.4.3-3.4.6) : demande envoyée
// (en_attente), devis reçu et son montant extrait (recu), puis, une fois la
// décision prise (syndic/IA, conseil syndical ou assemblée générale selon
// les seuils applicables), le devis choisi devient "retenu" et les autres
// (mise en concurrence, seuil B) "rejete".
const (
	DevisStatutEnAttente = "en_attente"
	DevisStatutRecu      = "recu"
	DevisStatutRetenu    = "retenu"
	DevisStatutRejete    = "rejete"
)

// DevisStatut est la table de référence du statut d'un Devis.
type DevisStatut struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Devis est une demande de devis adressée à un Prestataire pour
// l'intervention à réaliser dans le cadre d'un Ticket — un Ticket peut avoir
// plusieurs Devis (mise en concurrence, seuil B), d'où une table à part
// plutôt que des colonnes sur Incident (à la différence de Facture, qui
// n'existe qu'une fois par Ticket dans cette tranche).
//
// Les montants sont exprimés en centimes d'euro (entiers), conformément à la
// règle du projet — jamais en float.
type Devis struct {
	ID              int64
	CreatedAt       time.Time
	TicketID        int64 // FK -> ticket.id, NOT NULL
	PrestataireID   int64 // FK -> personne.id, NOT NULL
	MontantCentimes *int64
	StatutID        int64 // FK -> devis_statut.id, NOT NULL : à fixer explicitement à l'insertion
	DateDemande     *time.Time
	DateReception   *time.Time
	CreePar         *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
}
