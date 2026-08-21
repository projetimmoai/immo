package domain

import "time"

// Valeurs connues de conseil_syndical_mandat_statut.description.
const (
	ConseilSyndicalMandatStatutMembre       = "membre"
	ConseilSyndicalMandatStatutDemission    = "demission"
	ConseilSyndicalMandatStatutRevocation   = "revocation"
	ConseilSyndicalMandatStatutRemplacement = "remplacement"
)

// ConseilSyndicalMandatStatut est une table de référence (statut d'un
// mandat de membre du conseil syndical : membre, démission, révocation,
// remplacement).
type ConseilSyndicalMandatStatut struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// ConseilSyndicalMandat est le mandat d'une Personne comme membre du
// conseil syndical d'une Copropriete, sur une période donnée.
type ConseilSyndicalMandat struct {
	ID            int64
	CreatedAt     time.Time
	PersonneID    *int64     // FK -> personne.id
	CoproprieteID *int64     // FK -> copropriete.id
	DateDebut     *time.Time // date
	DateFin       *time.Time // date
	StatutID      *int64     // FK -> conseil_syndical_mandat_statut.id
	CreePar       *int64     // FK -> personne.id (gestionnaire à l'origine de la création)
}

// ConseilSyndicalPresident est le mandat d'une Personne comme président du
// conseil syndical d'une Copropriete, sur une période donnée. Élu soit en
// assemblée générale (EluEnAG), soit par le conseil syndical lui-même
// (EluParCS) — selon ce qu'autorise le règlement de copropriété.
type ConseilSyndicalPresident struct {
	ID            int64
	CreatedAt     time.Time
	PersonneID    *int64     // FK -> personne.id
	CoproprieteID *int64     // FK -> copropriete.id
	DateDebut     *time.Time // date
	DateFin       *time.Time // date
	EluEnAG       *bool
	EluParCS      *bool
	CreePar       *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
}
