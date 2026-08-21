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
// Roles liste tous les rôles applicables, jamais un seul. Lots et Contrats
// sont désormais toujours interrogés pour toute Personne connue (pas
// seulement quand un rôle laisse présager leur pertinence) : occupant,
// coproprietaire et prestataire sont dérivés de leur contenu plutôt que de
// booléens sur Personne/PersonneMorale, donc il faut les avoir en main
// avant même de calculer Roles (cf. rolesDe). CoproprietesGestion reste
// conditionné à RoleGestionnaire (booléen direct, pas de dérivation
// nécessaire).
type Contexte struct {
	Connu                       bool
	Personne                    *domain.Personne
	PersonnePhysique            *domain.PersonnePhysique         // renseigné si Personne.EstPhysique est vrai
	PersonneMorale              *domain.PersonneMorale           // renseigné si Personne.EstPhysique est faux
	Roles                       []domain.Role                    // tous les rôles applicables (occupant, coproprietaire, prestataire, gestionnaire, conseil_syndical) — peut être vide si aucun
	Lots                        []repository.LotAssocie          // tous les lots associés à Personne, quel que soit son rôle
	Coproprietes                []repository.CoproprieteAssociee // coproprietes des Lots ci-dessus, dédupliquées
	CoproprietesGestion         []repository.CoproprieteAssociee // coproprietes gérées par Personne (pertinent pour RoleGestionnaire)
	CoproprietesConseilSyndical []repository.CoproprieteAssociee // coproprietes où Personne a un mandat actif de membre du conseil syndical (pertinent pour RoleConseilSyndical) — toujours un sous-ensemble de Coproprietes, seul un copropriétaire pouvant y siéger
	Contrats                    []repository.ContratAssocie      // contrats où Personne (morale) est l'entreprise, avec leur copropriete ; non vide <=> RolePrestataire
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
	ListCoproprietesConseilSyndicalParPersonne(ctx context.Context, personneID int64) ([]repository.CoproprieteAssociee, error)
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

	// Occupant/coproprietaire n'étant plus des booléens sur Personne (cf.
	// domain.Role), il faut d'abord interroger les lots pour les déduire —
	// systématiquement, pas seulement quand un indice laissait présager un
	// rôle : un appel réseau de plus par e-mail, accepté au profit d'une
	// seule source de vérité (LotPersonneMap), sans risque de désync.
	lots, err := repo.ListLotsParPersonne(ctx, personne.ID)
	if err != nil {
		return nil, fmt.Errorf("email: recherche des lots associés à %s (personne id=%d): %w", adresseEmail, personne.ID, err)
	}
	c.Lots = lots
	c.Coproprietes = coproprietesDeLots(lots)

	// Idem pour prestataire (personne morale uniquement) : dérivé de
	// contrat.entreprise_id plutôt que d'un booléen sur personne_morale.
	if c.PersonneMorale != nil {
		contrats, err := repo.ListContratsParPrestataire(ctx, personne.ID)
		if err != nil {
			return nil, fmt.Errorf("email: recherche des contrats du prestataire %s (personne id=%d): %w", adresseEmail, personne.ID, err)
		}
		c.Contrats = contrats
	}

	c.Roles = rolesDe(personne, lots, c.Contrats)

	// Seul un copropriétaire (RoleCoproprietaire) peut siéger au conseil syndical :
	// pas la peine d'interroger conseil_syndical_mandat sinon.
	if c.ARole(domain.RoleCoproprietaire) {
		coproprietesCS, err := repo.ListCoproprietesConseilSyndicalParPersonne(ctx, personne.ID)
		if err != nil {
			return nil, fmt.Errorf("email: recherche des mandats de conseil syndical de %s (personne id=%d): %w", adresseEmail, personne.ID, err)
		}
		if len(coproprietesCS) > 0 {
			c.CoproprietesConseilSyndical = coproprietesCS
			c.Roles = append(c.Roles, domain.RoleConseilSyndical)
		}
	}

	if c.ARole(domain.RoleGestionnaire) {
		coproprietes, err := repo.ListCoproprietesParGestionnaire(ctx, personne.ID)
		if err != nil {
			return nil, fmt.Errorf("email: recherche des coproprietes gérées par %s (personne id=%d): %w", adresseEmail, personne.ID, err)
		}
		c.CoproprietesGestion = coproprietes
	}

	return c, nil
}

// rolesDe déduit tous les Role applicables à une Personne : RoleGestionnaire
// reste lu directement sur un booléen (Personne.EstGestionnaire) ;
// RoleOccupant/RoleCoproprietaire sont déduits de lots (au moins un lot où
// LotAssocie.EstOccupant/EstProprietaire est vrai) et RolePrestataire de
// contrats (non vide) — cf. domain.Role pour le détail de ce choix de
// conception. Peut retourner une liste vide (personne connue mais aucun
// rôle particulier).
func rolesDe(personne *domain.Personne, lots []repository.LotAssocie, contrats []repository.ContratAssocie) []domain.Role {
	var roles []domain.Role

	var estOccupant, estCoproprietaire bool
	for _, lot := range lots {
		if lot.EstOccupant != nil && *lot.EstOccupant {
			estOccupant = true
		}
		if lot.EstProprietaire != nil && *lot.EstProprietaire {
			estCoproprietaire = true
		}
	}
	if estOccupant {
		roles = append(roles, domain.RoleOccupant)
	}
	if estCoproprietaire {
		roles = append(roles, domain.RoleCoproprietaire)
	}
	if len(contrats) > 0 {
		roles = append(roles, domain.RolePrestataire)
	}
	if personne.EstGestionnaire != nil && *personne.EstGestionnaire {
		roles = append(roles, domain.RoleGestionnaire)
	}
	return roles
}

// coproprietesDeLots déduplique les coproprietes présentes dans une liste de
// LotAssocie (une même copropriete peut apparaître sur plusieurs lots),
// dans leur ordre de première apparition.
func coproprietesDeLots(lots []repository.LotAssocie) []repository.CoproprieteAssociee {
	var coproprietes []repository.CoproprieteAssociee
	vues := make(map[int64]bool, len(lots))
	for _, lot := range lots {
		if vues[lot.CoproprieteID] {
			continue
		}
		vues[lot.CoproprieteID] = true
		coproprietes = append(coproprietes, repository.CoproprieteAssociee{
			CoproprieteID:        lot.CoproprieteID,
			CoproprieteNom:       lot.CoproprieteNom,
			CoproprieteReference: lot.CoproprieteReference,
		})
	}
	return coproprietes
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
