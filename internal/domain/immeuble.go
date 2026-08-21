package domain

import "time"

// Copropriete représente un syndicat de copropriété. Le cabinet a deux
// fonctions commerciales indépendantes vis-à-vis d'une Copropriete : syndic
// de la copropriété entière (cf. EstSyndic), et/ou gestionnaire du lot d'un
// copropriétaire en gestion locative (cf. Personne.EstClient) — les deux
// mandats sont toujours rattachés à une Copropriete existante, jamais l'un
// sans l'autre.
type Copropriete struct {
	ID                                int64
	CreatedAt                         *time.Time
	SyndicID                          *int64 // FK -> personne.id (personne_morale du cabinet agissant comme syndic)
	EstSyndic                         *bool  // vrai si le cabinet est effectivement syndic de cette copropriete (par opposition à une copropriete simplement référencée)
	AdresseCodePostal                 *string
	AdresseVille                      *string
	AdressePaysCode                   *string
	AdresseRegion                     *string
	ExerciceDebut                     *time.Time // date
	ExerciceFin                       *time.Time // date
	AppelChargesFrequenceID           *int64     // FK -> frequence_id.id
	AppelChargesDate                  *time.Time // date
	AppelChargesNumJoursAvantEcheance *int64
	Nom                               *string
	ClesRepartition                   *float32
	ArreteComptableJour               *int64
	ArreteComptableMois               *int64
	NumeroImmatriculation             *string
	NumeroMandat                      *string
	MandatDateDebut                   *time.Time // date
	MandatDureeEnMois                 *int64
	CodeICS                           *string
	IBAN                              *string
	BIC                               *string
	MisAJour                          *time.Time
	CreePar                           *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
	Reference                         string // référence lisible (ex: "COP1"), générée par la base (DEFAULT), jamais fournie à l'insertion
}

// Batiment est un bâtiment physique appartenant à une Copropriete.
type Batiment struct {
	ID                int64
	NbLots            int32
	CreatedAt         *time.Time
	AdresseLigne1     *string
	AdresseLigne2     *string
	AdresseCodePostal *string
	AdresseVille      *string
	AdressePaysCode   *string
	Nom               *string
	AdresseRegion     *string
	CoproprieteID     *int64 // FK -> copropriete.id
	CreePar           *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
}

// Lot est un lot de copropriété (appartement, cave, parking...) au sein d'un Batiment.
type Lot struct {
	ID            int64
	CreatedAt     time.Time
	DesignationID *int64 // FK -> lot_designation.id
	Etage         *int64
	Tantiemes     *float32
	TypeAutre     *string
	BatimentID    *int64 // FK -> batiment.id
	EstSecondaire *bool
	Numero        *string
	EstActif      *bool
	CreePar       *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
	Reference     string // référence lisible (ex: "LOT1"), générée par la base (DEFAULT), jamais fournie à l'insertion
}

// LotDesignation est une table de référence (type de lot : appartement, cave, parking...).
type LotDesignation struct {
	ID          int64
	CreatedAt   time.Time
	Description *string
}

// LotPersonneMap relie un Lot à une Personne (propriétaire, gestionnaire, indivisaire...)
// sur une période donnée.
type LotPersonneMap struct {
	ID              int64
	CreatedAt       time.Time
	LotID           *int64 // FK -> lot.id
	PersonneID      *int64 // FK -> personne.id
	EstIndivision   *bool
	Debut           *time.Time // date
	Fin             *time.Time // date
	EstProprietaire *bool
	EstGestionnaire *bool
	CreePar         *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
}

// CoproprieteCollaborateurMap relie une Copropriete à une Personne collaborateur
// du cabinet de gestion (ex : gestionnaire en charge de cette copropriété).
type CoproprieteCollaborateurMap struct {
	ID            int64
	CreatedAt     time.Time
	CoproprieteID *int64 // FK -> copropriete.id
	PersonneID    *int64 // FK -> personne.id
	CreePar       *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
}
