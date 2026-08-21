package domain

// Role décrit un rôle qu'une Personne peut jouer vis-à-vis du cabinet de
// gestion — jamais codé en dur ailleurs que dans ce fichier. Une Personne
// peut cumuler plusieurs rôles à la fois (ex: copropriétaire ET occupant de
// son propre lot) : Role n'est pas exclusif.
//
// La plupart des rôles sont dérivés d'une relation existante plutôt que
// stockés directement sur Personne/PersonneMorale, pour éviter d'avoir
// deux sources de vérité à tenir synchronisées (cf. email.rolesDe pour le
// calcul) :
//   - RoleOccupant/RoleCoproprietaire : LotPersonneMap.EstOccupant/EstProprietaire
//     sur au moins un lot de la Personne.
//   - RoleFournisseur : au moins un contrat où PersonneMorale est l'entreprise
//     (contrat.entreprise_id).
//   - RoleConseilSyndical : un mandat actif (statut "membre") dans conseil_syndical_mandat.
//
// Seul RoleGestionnaire reste un attribut intrinsèque (Personne.EstGestionnaire)
// : rien ne permet de le déduire d'ailleurs (un collaborateur peut exister
// sans encore avoir de copropriete assignée dans copropriete_collaborateur_map).
type Role string

const (
	RoleOccupant        Role = "occupant"
	RoleCoproprietaire  Role = "coproprietaire" // jamais la copropriété elle-même : cf. Copropriete.EstSyndic (syndic) et gestion locative de son lot, deux fonctions commerciales indépendantes du cabinet
	RoleFournisseur     Role = "fournisseur"
	RoleGestionnaire    Role = "gestionnaire"     // Personne.EstGestionnaire (membre du cabinet de gestion)
	RoleConseilSyndical Role = "conseil_syndical" // mandat actif dans conseil_syndical_mandat (statut "membre") — s'ajoute à RoleCoproprietaire, ne le remplace pas : seul un copropriétaire peut siéger au conseil syndical
)

// CandidatCopropriete est une copropriété candidate pour le rattachement
// d'un e-mail à une copropriété précise, avec le ou les rôles sous
// lesquels elle est associée à l'expéditeur (une même copropriété peut
// être candidate sous plusieurs rôles, ex: coproprietaire ET gestionnaire
// de la même copropriété). Utilisé par internal/email (construction de la
// liste de candidats) et internal/claudeapi (entrée de la décision).
type CandidatCopropriete struct {
	CoproprieteID        int64
	CoproprieteNom       *string
	CoproprieteReference string
	Roles                []Role
}
