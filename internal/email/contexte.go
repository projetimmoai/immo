// Package email contient la logique métier de traitement des e-mails :
// enrichissement du contexte à partir de l'expéditeur (personne connue,
// lots, contrats...) et décision de routage à partir de ce contexte et du
// contenu du message — indépendamment de la source des messages (Gmail
// aujourd'hui via internal/gmailapi, potentiellement autre chose demain).
package email

import (
	"context"
	"fmt"

	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/gmailapi"
	"github.com/projetimmoai/immo/internal/repository"
)

// Contexte rassemble tout ce qu'on sait de l'expéditeur d'un e-mail, utile
// pour décider comment le router. Connu vaut false si l'adresse ne
// correspond à aucune Personne en base — ce n'est pas une situation
// anormale (expéditeur inconnu du système), les autres champs restent vides
// dans ce cas.
//
// Une Personne peut cumuler plusieurs Role à la fois (cf. domain.Role) :
// Roles liste tous les rôles applicables, jamais un seul. RolesCopropriete
// vient de la vue SQL personne_role (cf. repository.ListRolesParPersonne) —
// c'est elle qui dérive occupant/coproprietaire/prestataire/conseil_syndical
// (et lit gestionnaire/sys_admin/comptable/direction), la même vue que
// celle utilisée par les policies RLS : une seule dérivation, partagée.
// Lots/Contrats restent interrogés séparément : ils portent un détail
// (référence de lot, numéro de contrat...) que la vue ne porte pas, pas les
// rôles eux-mêmes.
type Contexte struct {
	Connu               bool
	Personne            *domain.Personne
	PersonnePhysique    *domain.PersonnePhysique          // renseigné si Personne.EstPhysique est vrai
	PersonneMorale      *domain.PersonneMorale            // renseigné si Personne.EstPhysique est faux
	Roles               []domain.Role                     // tous les rôles applicables (occupant, coproprietaire, prestataire, gestionnaire, conseil_syndical...) — peut être vide si aucun
	RolesCopropriete    []repository.PersonneRoleAssociee // détail rôle+copropriete (vue personne_role), utilisé pour un rattachement précis (cf. candidatsCoproprietes)
	Lots                []repository.LotAssocie           // tous les lots associés à Personne, quel que soit son rôle — détail (référence de lot...), pas la source de vérité des rôles
	CoproprietesGestion []repository.CoproprieteAssociee  // coproprietes gérées par Personne (copropriete_collaborateur_map) — routage informatif, pas un droit d'accès (cf. RLS : un gestionnaire voit toutes les coproprietes, pas seulement celles-ci)
	Contrats            []repository.ContratAssocie       // contrats où Personne (morale) est l'entreprise, avec leur copropriete ; non vide <=> RolePrestataire
}

// ARole indique si Roles contient le rôle donné.
func (c *Contexte) ARole(r domain.Role) bool {
	for _, role := range c.Roles {
		if role == r {
			return true
		}
	}
	return false
}

// contexteRepo est la portion de repository.Client utilisée ici — une
// interface étroite pour pouvoir tester avec des faux (cf. contexte_test.go).
type contexteRepo interface {
	FindPersonneByEmail(ctx context.Context, email string) (*domain.Personne, error)
	FindPersonnePhysiqueByPersonneID(ctx context.Context, personneID int64) (*domain.PersonnePhysique, error)
	FindPersonneMoraleByPersonneID(ctx context.Context, personneID int64) (*domain.PersonneMorale, error)
	ListLotsParPersonne(ctx context.Context, personneID int64) ([]repository.LotAssocie, error)
	ListContratsParPrestataire(ctx context.Context, entrepriseID int64) ([]repository.ContratAssocie, error)
	ListCoproprietesParGestionnaire(ctx context.Context, personneID int64) ([]repository.CoproprieteAssociee, error)
	ListRolesParPersonne(ctx context.Context, personneID int64) ([]repository.PersonneRoleAssociee, error)
}

// EnrichirExpediteur recherche l'expéditeur d'un e-mail par son adresse et
// rassemble tout son contexte métier (identité, rôles, lots, coproprietes,
// contrats) utile au routage.
func EnrichirExpediteur(ctx context.Context, repo contexteRepo, adresseEmail string) (*Contexte, error) {
	personne, err := repo.FindPersonneByEmail(ctx, adresseEmail)
	if err != nil {
		return nil, fmt.Errorf("email: recherche de l'expéditeur %s: %w", adresseEmail, err)
	}
	if personne == nil {
		return &Contexte{Connu: false}, nil
	}

	c := &Contexte{Connu: true, Personne: personne}

	switch {
	case personne.EstPhysique != nil && *personne.EstPhysique:
		pp, err := repo.FindPersonnePhysiqueByPersonneID(ctx, personne.ID)
		if err != nil {
			return nil, fmt.Errorf("email: recherche personne_physique pour %s (personne id=%d): %w", adresseEmail, personne.ID, err)
		}
		c.PersonnePhysique = pp

	case personne.EstPhysique != nil && !*personne.EstPhysique:
		pm, err := repo.FindPersonneMoraleByPersonneID(ctx, personne.ID)
		if err != nil {
			return nil, fmt.Errorf("email: recherche personne_morale pour %s (personne id=%d): %w", adresseEmail, personne.ID, err)
		}
		c.PersonneMorale = pm
	}

	// Détail des lots (référence, étage...) — pas la source de vérité des
	// rôles (cf. RolesCopropriete), mais utilisé plus loin dans le pipeline
	// (router.go) pour lister les lots concernés par la copropriete retenue.
	lots, err := repo.ListLotsParPersonne(ctx, personne.ID)
	if err != nil {
		return nil, fmt.Errorf("email: recherche des lots associés à %s (personne id=%d): %w", adresseEmail, personne.ID, err)
	}
	c.Lots = lots

	// Idem pour les contrats (personne morale uniquement) : détail (numéro
	// de contrat...), pas la source de vérité de RolePrestataire.
	if c.PersonneMorale != nil {
		contrats, err := repo.ListContratsParPrestataire(ctx, personne.ID)
		if err != nil {
			return nil, fmt.Errorf("email: recherche des contrats du prestataire %s (personne id=%d): %w", adresseEmail, personne.ID, err)
		}
		c.Contrats = contrats
	}

	rolesCopropriete, err := repo.ListRolesParPersonne(ctx, personne.ID)
	if err != nil {
		return nil, fmt.Errorf("email: recherche des rôles de %s (personne id=%d): %w", adresseEmail, personne.ID, err)
	}
	c.RolesCopropriete = rolesCopropriete
	c.Roles = rolesDistincts(rolesCopropriete)

	// CoproprietesGestion reste une requête séparée : elle vient de
	// copropriete_collaborateur_map (assignation de routage), pas de
	// personne_role (qui ne scope pas gestionnaire à une copropriete en
	// particulier — cf. RLS, un gestionnaire a accès à tout).
	if c.ARole(domain.RoleGestionnaire) {
		coproprietes, err := repo.ListCoproprietesParGestionnaire(ctx, personne.ID)
		if err != nil {
			return nil, fmt.Errorf("email: recherche des coproprietes gérées par %s (personne id=%d): %w", adresseEmail, personne.ID, err)
		}
		c.CoproprietesGestion = coproprietes
	}

	return c, nil
}

// rolesDistincts déduplique les Role présents dans une liste de
// PersonneRoleAssociee (une même Personne peut avoir plusieurs lignes pour
// un même rôle, une par copropriete concernée), dans leur ordre de première
// apparition.
func rolesDistincts(lignes []repository.PersonneRoleAssociee) []domain.Role {
	var roles []domain.Role
	vus := make(map[domain.Role]bool, len(lignes))
	for _, l := range lignes {
		r := domain.Role(l.Role)
		if vus[r] {
			continue
		}
		vus[r] = true
		roles = append(roles, r)
	}
	return roles
}

// TraiterMessage enrichit le contexte de l'expéditeur d'un Message Gmail
// normalisé (internal/gmailapi). Ne modifie rien en base ni ailleurs.
//
// Ne va pas plus loin (détermination de la copropriété, puis routage) :
// ces étapes suivantes ont chacune leurs propres dépendances (client
// Claude dédié, dépôt pour la journalisation) qui ne se prêtent pas à un
// simple enchaînement ici — c'est au code appelant (worker, à venir)
// d'orchestrer DetermineCopropriete puis le routage par rôle (à venir).
func TraiterMessage(ctx context.Context, repo contexteRepo, msg *gmailapi.Message) (*Contexte, error) {
	return EnrichirExpediteur(ctx, repo, msg.From)
}
