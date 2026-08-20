package domain

import "time"

// Valeurs connues de categorie_email.description (voir migration 001).
// Ce sont des tables de référence éditables en base : ces constantes servent
// de clés de recherche (par description), jamais d'ID en dur.
const (
	CategorieEmailSinistre          = "sinistre"
	CategorieEmailIncident          = "incident"
	CategorieEmailAssembleeGenerale = "assemblee_generale"
	CategorieEmailAutre             = "autre"
	CategorieEmailIndetermine       = "indetermine"
)

// Valeurs connues de email_statut_traitement.description.
const (
	EmailStatutNouveau   = "nouveau"
	EmailStatutClassifie = "classifie"
	EmailStatutTraite    = "traite"
	EmailStatutIgnore    = "ignore"
	EmailStatutErreur    = "erreur"
)

// CategorieEmail est une table de référence (sinistre, incident, AG, autre...).
type CategorieEmail struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// EmailStatutTraitement est une table de référence pour le statut de
// traitement d'un Email par le worker (nouveau, classifie, traite...).
type EmailStatutTraitement struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// Email est le journal de tout e-mail reçu par le worker, traité ou non.
type Email struct {
	ID                   int64
	CreatedAt            time.Time
	MessageID            *string // header RFC822 Message-ID, sert à dédupliquer
	DateReception        time.Time
	ExpediteurEmail      string
	ExpediteurPersonneID *int64 // FK -> personne.id, résolu si l'expéditeur est connu
	Objet                *string
	CorpsTexte           *string
	CorpsHTML            *string
	CategorieID          *int64 // FK -> categorie_email.id, nul tant que non classifié
	CoproprieteID        *int64 // FK -> copropriete.id, résolu si identifiable
	LotID                *int64 // FK -> lot.id, résolu si identifiable
	StatutTraitementID   int64  // FK -> email_statut_traitement.id, NOT NULL : à fixer explicitement à l'insertion (pas de DEFAULT en base)
	TraiteLe             *time.Time
	TraitePar            *int64 // FK -> personne.id (collaborateur)
	ErreurTraitement     *string
}

// EmailPieceJointe est une pièce jointe d'un Email, stockée dans un service
// externe (Google Drive ou autre) et référencée ici par son URL.
type EmailPieceJointe struct {
	ID           int64
	CreatedAt    time.Time
	EmailID      int64
	NomFichier   string
	TypeMime     *string
	TailleOctets *int64
	URL          *string
}
