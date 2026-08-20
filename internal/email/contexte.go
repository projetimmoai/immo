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
// Roles liste tous les rôles applicables, jamais un seul. Lots,
// Coproprietes et CoproprietesGestion rassemblent les coproprietes
// concernées (pour occupant/client via les lots, pour gestionnaire via
// copropriete_collaborateur_map) ; Contrats les contrats fournisseur.
type Contexte struct {
	Connu               bool
	Personne            *domain.Personne
	PersonnePhysique    *domain.PersonnePhysique         // renseigné si Personne.EstPhysique est vrai
	PersonneMorale      *domain.PersonneMorale           // renseigné si Personne.EstPhysique est faux
	Roles               []domain.Role                    // tous les rôles applicables (occupant, client, fournisseur, gestionnaire) — peut être vide si aucun
	Lots                []repository.LotAssocie          // lots où Personne est propriétaire, gestionnaire de lot ou indivisaire (pertinent pour RoleOccupant/RoleClient)
	Coproprietes        []repository.CoproprieteAssociee // coproprietes des Lots ci-dessus, dédupliquées (pertinent pour RoleOccupant/RoleClient)
	CoproprietesGestion []repository.CoproprieteAssociee // coproprietes gérées par Personne (pertinent pour RoleGestionnaire)
	Contrats            []repository.ContratAssocie      // contrats où Personne est le fournisseur, avec leur copropriete (pertinent pour RoleFournisseur)
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
	ListContratsParFournisseur(ctx context.Context, entrepriseID int64) ([]repository.ContratAssocie, error)
	ListCoproprietesParGestionnaire(ctx context.Context, personneID int64) ([]repository.CoproprieteAssociee, error)
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

	c.Roles = rolesDe(personne, c.PersonneMorale)

	if c.ARole(domain.RoleFournisseur) {
		contrats, err := repo.ListContratsParFournisseur(ctx, personne.ID)
		if err != nil {
			return nil, fmt.Errorf("email: recherche des contrats du fournisseur %s (personne id=%d): %w", adresseEmail, personne.ID, err)
		}
		c.Contrats = contrats
	}

	if c.ARole(domain.RoleOccupant) || c.ARole(domain.RoleClient) {
		lots, err := repo.ListLotsParPersonne(ctx, personne.ID)
		if err != nil {
			return nil, fmt.Errorf("email: recherche des lots associés à %s (personne id=%d): %w", adresseEmail, personne.ID, err)
		}
		c.Lots = lots
		c.Coproprietes = coproprietesDeLots(lots)
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

// rolesDe déduit tous les Role applicables à une Personne à partir de ses
// booléens (et de ceux de sa PersonneMorale le cas échéant, pour
// RoleFournisseur). Peut retourner une liste vide (personne connue mais
// aucun rôle particulier renseigné).
func rolesDe(personne *domain.Personne, personneMorale *domain.PersonneMorale) []domain.Role {
	var roles []domain.Role
	if personne.EstOccupant != nil && *personne.EstOccupant {
		roles = append(roles, domain.RoleOccupant)
	}
	if personne.EstClient != nil && *personne.EstClient {
		roles = append(roles, domain.RoleClient)
	}
	if personneMorale != nil && personneMorale.EstFournisseur != nil && *personneMorale.EstFournisseur {
		roles = append(roles, domain.RoleFournisseur)
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
// normalisé (internal/gmailapi) et décide de son routage (cf. DecideRoute),
// en une seule étape. Ne modifie rien en base ni ailleurs : c'est au code
// appelant (worker, à venir) de décider quoi faire du résultat (créer un
// incident, un sinistre, l'ignorer...).
func TraiterMessage(ctx context.Context, repo contexteRepo, msg *gmailapi.Message) (*Contexte, Decision, error) {
	contexte, err := EnrichirExpediteur(ctx, repo, msg.From)
	if err != nil {
		return nil, Decision{}, err
	}
	decision := DecideRoute(contexte, msg.Subject, msg.BodyText)
	return contexte, decision, nil
}
