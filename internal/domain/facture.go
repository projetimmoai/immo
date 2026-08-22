package domain

import "time"

// Valeurs connues de facture_statut.description — cycle de vie d'une
// Facture (cf. docs/cycle-vie-incident.md, phase 5.5) : réception,
// validation du montant face au devis/contrat, mise en paiement (gatée sur
// une vérification positive de l'incident, cf. Incident.
// VerificationResultatID — un travail mal exécuté ne doit pas être payé,
// exception d'inexécution), puis rapprochement comptable.
const (
	FactureStatutEnAttente  = "en_attente"
	FactureStatutRecue      = "recue"
	FactureStatutValidee    = "validee"
	FactureStatutPayee      = "payee"
	FactureStatutRapprochee = "rapprochee"
)

// FactureStatut est la table de référence du statut d'une Facture.
type FactureStatut struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Facture est la facture d'un Prestataire pour l'intervention réalisée dans
// le cadre d'un Ticket. Rattachée à Ticket (pas à Incident) : une facture de
// prestataire concernera un jour d'autres types de ticket (travaux).
//
// Les montants sont exprimés en centimes d'euro (entiers), conformément à la
// règle du projet — jamais en float.
type Facture struct {
	ID                     int64
	CreatedAt              time.Time
	TicketID               int64 // FK -> ticket.id, NOT NULL
	PrestataireID          int64 // FK -> personne.id, NOT NULL
	MontantFactureCentimes int64
	StatutID               int64 // FK -> facture_statut.id, NOT NULL : à fixer explicitement à l'insertion
	DateReception          *time.Time
	DateValidation         *time.Time
	ValidePar              *int64 // FK -> personne.id (gestionnaire ayant validé le montant)
	DatePaiement           *time.Time
	PayePar                *int64 // FK -> personne.id (gestionnaire ayant déclenché le paiement)
	DateRapprochement      *time.Time
	RapprochePar           *int64 // FK -> personne.id (gestionnaire ayant fait le rapprochement comptable)
	CreePar                *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
}
