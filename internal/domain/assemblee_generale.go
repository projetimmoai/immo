package domain

import "time"

// Valeurs connues de ag_resultat.description — résultat du vote d'une
// assemblée générale sur une résolution (cf. Incident.AGResultatID, phase
// 3.4.21-3.4.22 du graphe de cycle de vie d'un incident). Pas encore de
// table dédiée pour l'assemblée générale elle-même (convocations, ordre du
// jour, autres résolutions...) : seul ce résultat, rattaché directement à
// Incident, est modélisé pour l'instant — cf. domain.ExerciceComptable.
// AgApprobationID, qui note la même limitation ailleurs dans le projet.
const (
	AGResultatApprouve = "approuve"
	AGResultatRejete   = "rejete"
)

// AGResultat est la table de référence du résultat du vote d'une résolution
// en assemblée générale.
type AGResultat struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// ConseilSyndicalDelegation est l'enveloppe C du graphe de cycle de vie d'un
// incident (cf. docs/cycle-vie-incident.md, phase 0.C ; art. 21-1 à 21-5 de
// la loi du 10 juillet 1965) : une délégation de pouvoir de l'assemblée
// générale au conseil syndical pour décider des dépenses d'entretien
// courant, dans la limite de PlafondCentimes, pour une durée maximale de
// deux ans (DateDebut/DateFin). Tant qu'elle est active et couvre le
// montant, le conseil syndical vote et choisit lui-même l'artisan — le
// syndic/IA n'est qu'exécutant et payeur (par opposition au seuil A, simple
// avis consultatif).
type ConseilSyndicalDelegation struct {
	ID              int64
	CreatedAt       time.Time
	CoproprieteID   int64 // FK -> copropriete.id, NOT NULL
	PlafondCentimes int64
	DateDebut       time.Time
	DateFin         time.Time
	DateVoteAG      *time.Time
	CreePar         *int64 // FK -> personne.id (gestionnaire à l'origine de la création)
}
