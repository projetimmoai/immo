package domain

import "time"

// Personne est l'entité pivot commune à toute personne physique ou morale
// connue du système (propriétaire, locataire, collaborateur, fournisseur...).
// Selon EstPhysique, elle est complétée par une ligne PersonnePhysique ou
// PersonneMorale.
type Personne struct {
	ID                int64
	CreatedAt         time.Time
	Tel               *string
	AdresseLigne1     *string
	AdresseLigne2     *string
	AdresseVille      *string
	AdresseRegion     *string
	AdresseCodePostal *string
	AdressePaysCode   *string
	Email             *string
	EstPhysique       *bool
	Reference         string // référence lisible (ex: "PER1"), générée par la base (DEFAULT), jamais fournie à l'insertion
	IBAN              *string
	BIC               *string
	EstGestionnaire   *bool // membre du cabinet de gestion
	EstOccupant       *bool // occupe un lot (locataire ou propriétaire occupant)
	EstClient         *bool
	CreePar           *int64 // FK -> personne.id (gestionnaire à l'origine de la création ; auto-référence)
}

// PersonnePhysique complète une Personne quand EstPhysique est vrai.
type PersonnePhysique struct {
	ID         int64
	CreatedAt  time.Time
	Nom        *string
	Prenom     *string
	PersonneID *int64 // FK -> personne.id
	GenreID    *int64 // FK -> personne_physique_genre.id
}

// PersonnePhysiqueGenre est une table de référence (genre de la personne physique).
type PersonnePhysiqueGenre struct {
	ID          int64
	CreatedAt   time.Time
	Description *string
}

// PersonneMorale complète une Personne quand EstPhysique est faux.
type PersonneMorale struct {
	ID                int64
	CreatedAt         time.Time
	Nom               *string
	EmailFactures     *string
	EstCabinetGestion *bool
	FormeJuridiqueID  *int64 // FK -> personne_morale_forme_juridique.id
	PersonneID        *int64 // FK -> personne.id
	EstFournisseur    *bool
}

// PersonneMoraleFormeJuridique est une table de référence (SCI, SARL, etc.).
type PersonneMoraleFormeJuridique struct {
	ID          int64
	CreatedAt   time.Time
	Description *string
}
