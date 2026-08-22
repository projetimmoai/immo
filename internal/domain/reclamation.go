package domain

import "time"

// Valeurs connues de reclamation_statut.description — cycle de vie d'une
// Reclamation (cf. docs/cycle-vie-incident.md, phase 5.3) : envoyée au
// prestataire après une vérification négative, puis soit acceptée (le
// prestataire revient corriger, retour phase 4 du graphe), soit refusée —
// un litige, hors du périmètre de ce graphe (traité séparément).
const (
	ReclamationStatutEnvoyee  = "envoyee"
	ReclamationStatutAcceptee = "acceptee"
	ReclamationStatutRefusee  = "refusee"
)

// ReclamationStatut est la table de référence du statut d'une Reclamation.
type ReclamationStatut struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Reclamation est une réclamation adressée à un Prestataire lorsque la
// vérification d'un Incident s'avère négative (occupant non satisfait, ou
// constat du gestionnaire humain sur place) — cf. Incident.
// VerificationResultatID. Rattachée à Ticket (pas à Incident) : un même
// ticket peut donner lieu à plusieurs réclamations successives si le
// prestataire revient corriger mais que le problème persiste encore.
type Reclamation struct {
	ID            int64
	CreatedAt     time.Time
	TicketID      int64 // FK -> ticket.id, NOT NULL
	PrestataireID int64 // FK -> personne.id, NOT NULL
	Texte         string
	StatutID      int64 // FK -> reclamation_statut.id, NOT NULL : à fixer explicitement à l'insertion
	DateEnvoi     *time.Time
	DateReponse   *time.Time
	ReponseTexte  *string
	CreePar       *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
}
