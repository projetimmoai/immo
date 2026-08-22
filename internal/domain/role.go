package domain

// Role décrit un rôle qu'une Personne peut jouer vis-à-vis du cabinet de
// gestion — jamais codé en dur ailleurs que dans ce fichier, et jamais avec
// une valeur différente du contenu de la table de référence "role" en base
// (les deux doivent rester le même vocabulaire).
// Une Personne peut cumuler plusieurs rôles à la fois (ex: copropriétaire
// ET occupant de son propre lot) : Role n'est pas exclusif.
//
// La table role en base ne stocke que le vocabulaire (les 10 noms
// possibles) — jamais l'attribution "quelle Personne a quel rôle" : ça
// resterait une deuxième source de vérité à tenir synchronisée avec les
// tables qui portent déjà cette information. L'attribution effective est
// calculée par la vue personne_role (cf. repository.ListRolesParPersonne),
// qui unifie :
//   - RoleOccupant/RoleCoproprietaire : LotPersonneMap.EstOccupant/EstProprietaire
//     sur au moins un lot de la Personne.
//   - RolePrestataire : au moins un contrat où PersonneMorale est l'entreprise
//     (contrat.entreprise_id).
//   - RoleConseilSyndical : un mandat actif (statut "membre") dans conseil_syndical_mandat.
//   - RoleGestionnaire/RoleSysAdmin/RoleComptable/RoleDirection : attributs
//     intrinsèques (Personne.EstGestionnaire/EstSysAdmin/EstComptable/EstDirigeant)
//     — rien ne permet de les déduire d'une autre table.
//
// RoleBailleur/RoleLocataire existent dans le vocabulaire mais n'ont
// aujourd'hui aucune attribution possible : la gestion locative (qui loue
// quel lot, à qui) n'est pas encore modélisée.
type Role string

const (
	RoleOccupant        Role = "occupant"
	RoleCoproprietaire  Role = "coproprietaire" // jamais la copropriété elle-même : cf. Copropriete.EstSyndic (syndic) et gestion locative de son lot, deux fonctions commerciales indépendantes du cabinet
	RolePrestataire     Role = "prestataire"
	RoleGestionnaire    Role = "gestionnaire"     // Personne.EstGestionnaire (membre du cabinet de gestion)
	RoleConseilSyndical Role = "conseil_syndical" // mandat actif dans conseil_syndical_mandat (statut "membre") — s'ajoute à RoleCoproprietaire, ne le remplace pas : seul un copropriétaire peut siéger au conseil syndical
	RoleSysAdmin        Role = "sys_admin"        // Personne.EstSysAdmin — administration technique, pas la gestion métier (cf. RoleDirection)
	RoleComptable       Role = "comptable"        // Personne.EstComptable — membre du cabinet en charge de la comptabilité
	RoleDirection       Role = "direction"        // Personne.EstDirigeant — direction du cabinet
	RoleBailleur        Role = "bailleur"         // pas encore attribuable : gestion locative non modélisée
	RoleLocataire       Role = "locataire"        // pas encore attribuable : gestion locative non modélisée
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
