package domain

// Role décrit un rôle qu'une Personne peut jouer vis-à-vis du cabinet de
// gestion, déduit des booléens des tables personne/personne_morale — jamais
// codé en dur ailleurs que dans ce fichier. Une Personne peut cumuler
// plusieurs rôles à la fois (ex: copropriétaire ET occupant de son propre
// lot) : Role n'est pas exclusif, cf. Personne.EstOccupant, EstClient,
// EstGestionnaire et PersonneMorale.EstFournisseur.
type Role string

const (
	RoleOccupant     Role = "occupant"     // Personne.EstOccupant
	RoleClient       Role = "client"       // Personne.EstClient — copropriétaire (Copropriete.EstSyndic) ou propriétaire en gestion locative (sans lien avec une Copropriete)
	RoleFournisseur  Role = "fournisseur"  // PersonneMorale.EstFournisseur
	RoleGestionnaire Role = "gestionnaire" // Personne.EstGestionnaire (membre du cabinet de gestion)
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
