package domain

import "time"

// Personne est l'entité pivot commune à toute personne physique ou morale
// connue du système (propriétaire, locataire, collaborateur, fournisseur...).
// Selon EstPhysique, elle est complétée par une ligne PersonnePhysique ou
// PersonneMorale.
//
// Occupant et coproprietaire ne sont volontairement pas des champs ici :
// ce sont des rôles dérivés de LotPersonneMap (est_occupant/est_proprietaire
// sur chaque lot associé), pas des attributs intrinsèques de la personne —
// cf. domain.Role et email.rolesDe, qui les calculent à partir des lots
// plutôt que de dupliquer l'information dans une colonne à tenir
// synchronisée (risque de désynchronisation écarté sciemment).
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
	EstGestionnaire   *bool  // membre du cabinet de gestion — pas dérivable d'ailleurs (un collaborateur peut exister sans copropriete assignée), reste un attribut propre
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
//
// Fournisseur n'est volontairement pas un champ ici : c'est un rôle dérivé
// de contrat.entreprise_id (au moins un contrat où cette personne morale
// est l'entreprise), pas un attribut intrinsèque — même raisonnement que
// pour les rôles occupant/coproprietaire d'une Personne, dérivés de
// LotPersonneMap (cf. sa doc).
type PersonneMorale struct {
	ID                int64
	CreatedAt         time.Time
	Nom               *string
	EmailFactures     *string
	EstCabinetGestion *bool
	FormeJuridiqueID  *int64 // FK -> personne_morale_forme_juridique.id
	PersonneID        *int64 // FK -> personne.id
}

// PersonneMoraleFormeJuridique est une table de référence (SCI, SARL, etc.).
type PersonneMoraleFormeJuridique struct {
	ID          int64
	CreatedAt   time.Time
	Description *string
}
