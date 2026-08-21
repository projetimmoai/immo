package domain

import "time"

// Valeurs connues de log_type.description. Ce sont des clés de recherche
// (par description), jamais des ID en dur.
const (
	LogTypeCoproprieteNonIdentifiee      = "copropriete_non_identifiee"
	LogTypeExpediteurInconnu             = "expediteur_inconnu"
	LogTypeActionIndeterminee            = "action_indeterminee"
	LogTypeConfianceFaible               = "confiance_faible"
	LogTypeErreurAppelClaude             = "erreur_appel_claude"
	LogTypeDesynchronisationBaseStockage = "desynchronisation_base_stockage"
)

// LogType est une table de référence pour les types d'évènements consignés
// dans Log.
type LogType struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Log est un évènement consigné — le plus souvent une anomalie ou une
// erreur rencontrée pendant le traitement d'un e-mail (cf. LogType*) —,
// avec assez de contexte pour investiguer : l'e-mail, la copropriete et/ou
// la personne concernés, quand ils sont connus.
type Log struct {
	ID            int64
	CreatedAt     time.Time
	LogTypeID     int64 // FK -> log_type.id, NOT NULL : à fixer explicitement à l'insertion
	Message       *string
	EmailID       *int64 // FK -> email.id, si l'évènement est lié au traitement d'un e-mail
	CoproprieteID *int64 // FK -> copropriete.id, si identifiable
	PersonneID    *int64 // FK -> personne.id, si l'expéditeur ou une personne concernée est connue
}
