package domain

import "time"

// Copropriete représente un syndicat de copropriété. Le cabinet a deux
// fonctions commerciales indépendantes vis-à-vis d'une Copropriete : syndic
// de la copropriété entière (cf. EstSyndic), et/ou gestionnaire du lot d'un
// copropriétaire en gestion locative (rôle Coproprietaire, cf. Role) — les
// deux mandats sont toujours rattachés à une Copropriete existante, jamais
// l'un sans l'autre.
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
	MisAJour                          *time.Time
	CreePar                           *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
	Reference                         string // référence lisible (ex: "COP1"), générée par la base (DEFAULT), jamais fournie à l'insertion
	BanqueID                          *int64 // FK -> banque.id (nullable, 1-1)

	// PlafondOrdreServiceCentimes est le seuil D du graphe de cycle de vie
	// d'un incident (cf. docs/cycle-vie-incident.md, phase 0.D) : en-dessous,
	// le prestataire retenu intervient directement, sans devis préalable.
	// Fixé par le gestionnaire ou la convention de gestion — pas d'origine
	// légale, par opposition aux seuils A/B et au pouvoir ordinaire du
	// syndic ci-dessous (origine légale ou votée en AG), et à l'enveloppe C
	// (délégation CS, cf. domain.ConseilSyndicalDelegation).
	PlafondOrdreServiceCentimes *int64

	// SeuilAConsultationCSCentimes est le seuil A (phase 0.A, art. 21 al. 2
	// de la loi du 10 juillet 1965) : au-delà, le syndic doit recueillir
	// l'avis écrit du conseil syndical avant d'engager la dépense — avis
	// consultatif, le syndic reste décisionnaire.
	SeuilAConsultationCSCentimes *int64

	// SeuilBMiseEnConcurrenceCentimes est le seuil B (phase 0.B, art. 21
	// al. 3) : au-delà, le syndic doit présenter au moins deux devis
	// d'entreprises distinctes plutôt qu'un seul.
	SeuilBMiseEnConcurrenceCentimes *int64

	// SeuilPouvoirSyndicCentimes est le seuil au-delà duquel même l'avis du
	// conseil syndical ne suffit plus : la dépense excède le pouvoir
	// ordinaire du syndic et doit être votée en assemblée générale (phase
	// 3.4.17-3.4.22 du graphe) — distinct de l'enveloppe C (délégation
	// explicite au CS, cf. domain.ConseilSyndicalDelegation), qui court-
	// circuite ce seuil tant qu'elle est active et couvre le montant.
	SeuilPouvoirSyndicCentimes *int64
}

// Banque contient des informations bancaires (IBAN, BIC...) — dans une
// table séparée des entités qui les possèdent (pas des colonnes en plus sur
// Copropriete/Personne) précisément pour que le RLS puisse les verrouiller
// indépendamment : RLS filtre des lignes, jamais des colonnes, donc une
// donnée sensible mélangée à des colonnes publiques dans la même table ne
// peut pas être protégée par policy.
//
// Table partagée par plusieurs entités : Copropriete (accès dirigeant/
// sys_admin uniquement) et Personne (accès dirigeant/sys_admin, plus la
// personne elle-même pour sa propre ligne) — chacune pointe vers une ligne
// de Banque via son propre BanqueID, jamais l'inverse.
type Banque struct {
	ID      int64 // PK propre
	CodeICS *string
	IBAN    *string
	BIC     *string
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

// LotPersonneMap relie un Lot à une Personne (propriétaire, occupant,
// gestionnaire, indivisaire...) sur une période donnée. EstProprietaire et
// EstOccupant sont la source de vérité des rôles domain.RoleCoproprietaire
// et domain.RoleOccupant (cf. email.rolesDe) — pas de booléen redondant sur
// Personne.
type LotPersonneMap struct {
	ID              int64
	CreatedAt       time.Time
	LotID           *int64 // FK -> lot.id
	PersonneID      *int64 // FK -> personne.id
	EstIndivision   *bool
	Debut           *time.Time // date
	Fin             *time.Time // date
	EstProprietaire *bool
	EstOccupant     *bool
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
