package domain

// Role décrit un rôle qu'une Personne peut jouer vis-à-vis du cabinet de
// gestion, déduit des booléens des tables personne/personne_morale (ou,
// pour RoleConseilSyndical, d'un mandat actif dans conseil_syndical_mandat)
// — jamais codé en dur ailleurs que dans ce fichier. Une Personne peut
// cumuler plusieurs rôles à la fois (ex: copropriétaire ET occupant de son
// propre lot) : Role n'est pas exclusif, cf. Personne.EstOccupant,
// EstCoproprietaire, EstGestionnaire et PersonneMorale.EstFournisseur.
type Role string

const (
	RoleOccupant        Role = "occupant"         // Personne.EstOccupant
	RoleClient          Role = "client"           // Personne.EstCoproprietaire — jamais la copropriété elle-même : cf. Copropriete.EstSyndic (syndic) et gestion locative de son lot, deux fonctions commerciales indépendantes du cabinet
	RoleFournisseur     Role = "fournisseur"      // PersonneMorale.EstFournisseur
	RoleGestionnaire    Role = "gestionnaire"     // Personne.EstGestionnaire (membre du cabinet de gestion)
	RoleConseilSyndical Role = "conseil_syndical" // mandat actif dans conseil_syndical_mandat (statut "membre") — s'ajoute à RoleClient, ne le remplace pas : seul un copropriétaire peut siéger au conseil syndical
)

// CandidatCopropriete est une copropriété candidate pour le rattachement
// d'un e-mail à une copropriété précise, avec le ou les rôles sous
// lesquels elle est associée à l'expéditeur (une même copropriété peut
// être candidate sous plusieurs rôles, ex: client ET gestionnaire de la
// même copropriété). Utilisé par internal/email (construction de la liste
// de candidats) et internal/claudeapi (entrée de la décision).
type CandidatCopropriete struct {
	CoproprieteID        int64
	CoproprieteNom       *string
	CoproprieteReference string
	Roles                []Role
}
