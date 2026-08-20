package domain

// ContexteRoutage rassemble ce qu'on sait pour router un e-mail une fois sa
// copropriété identifiée (cf. email.DetermineCopropriete) : la personne
// expéditrice et son rôle, la copropriété retenue, et — quand ils sont
// déterminables pour cette copropriété précise — les références des lots
// (occupant/client) et les numéros de contrat (fournisseur) de cette
// personne. Utilisé par internal/email (construction, dispatch vers les
// fonctions de traitement par action) et internal/claudeapi (entrée de
// DecideAction).
type ContexteRoutage struct {
	Personne             *Personne
	Role                 *Role // rôle retenu par DetermineCopropriete ; nil si non déterminé
	CoproprieteID        int64
	CoproprieteReference string
	LotsReferences       []string // références des lots de la Personne dans cette Copropriete, si occupant/client
	ContratsNumeros      []string // numéros des contrats de la Personne dans cette Copropriete, si fournisseur
}
