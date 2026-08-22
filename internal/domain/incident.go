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

// Valeurs connues de mode_verification.description — comment on s'assure
// qu'un incident est réellement résolu avant de le refermer (cf. Incident.
// ModeVerificationID) : demander confirmation à l'occupant (cas courant),
// vérification physique par un gestionnaire humain (rapport du prestataire
// incertain, enjeu le justifiant), ou aucune vérification supplémentaire
// (problème bénin, rapport du prestataire déjà sans ambiguïté).
const (
	ModeVerificationConfirmationOccupant = "confirmation_occupant"
	ModeVerificationGH                   = "verification_gh"
	ModeVerificationJugeeInutile         = "jugee_inutile"
)

// ModeVerification est la table de référence du mode de vérification retenu
// pour un Incident.
type ModeVerification struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Valeurs connues de verification_resultat.description — résultat de la
// vérification retenue (cf. Incident.VerificationResultatID). "jugee_inutile"
// (cf. ModeVerification*) vaut automatiquement Positive : aucune vérification
// n'a été jugée nécessaire, ce n'est pas un résultat en soi.
const (
	VerificationResultatPositive = "positive"
	VerificationResultatNegative = "negative"
)

// VerificationResultat est la table de référence du résultat de la
// vérification d'un Incident.
type VerificationResultat struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Incident complète un Ticket (cf. domain.Ticket, qui porte le squelette
// commun : statut, déclarant, copropriete/lot, e-mail d'origine...) avec les
// champs propres à un dysfonctionnement technique ponctuel — par opposition
// à un sinistre (dégât des eaux, incendie...) ou des travaux déjà planifiés.
//
// PrestataireID/MontantEstimeCentimes/RapportIntervention/ModeVerification*/
// VerificationResultat* couvrent la tranche verticale "cas simple, montant
// sous le plafond D" du graphe de cycle de vie (cf. docs/cycle-vie-incident.
// md) : sélection du prestataire, suivi de l'intervention, puis
// vérification — sans devis, décision du conseil syndical déléguée ou avis,
// ni vote en assemblée générale, laissés pour un prochain jalon.
type Incident struct {
	TicketID               int64  // PK -> ticket.id (1-1, pas d'id propre)
	CategorieTechniqueID   *int64 // FK -> categorie_technique.id
	UrgenceID              *int64 // FK -> niveau_urgence.id
	DateResolution         *time.Time
	PrestataireID          *int64 // FK -> personne.id (personne_morale prestataire retenu)
	MontantEstimeCentimes  *int64 // estimation indicative (phase 2.2), jamais un montant engageant
	RapportIntervention    *string
	ModeVerificationID     *int64 // FK -> mode_verification.id
	VerificationResultatID *int64 // FK -> verification_resultat.id
}
