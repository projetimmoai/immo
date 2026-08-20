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
type Contexte struct {
	Connu            bool
	Personne         *domain.Personne
	PersonnePhysique *domain.PersonnePhysique // renseigné si Personne.EstPhysique est vrai
	PersonneMorale   *domain.PersonneMorale   // renseigné si Personne.EstPhysique est faux
	Lots             []repository.LotAssocie  // lots où Personne est propriétaire, gestionnaire ou indivisaire
	Contrats         []*domain.Contrat        // contrats où Personne est le fournisseur (renseigné si EstFournisseur)
}

// contexteRepo est la portion de repository.Client utilisée ici — une
// interface étroite pour pouvoir tester avec des faux (cf. contexte_test.go).
type contexteRepo interface {
	FindPersonneByEmail(ctx context.Context, email string) (*domain.Personne, error)
	FindPersonnePhysiqueByPersonneID(ctx context.Context, personneID int64) (*domain.PersonnePhysique, error)
	FindPersonneMoraleByPersonneID(ctx context.Context, personneID int64) (*domain.PersonneMorale, error)
	ListLotsParPersonne(ctx context.Context, personneID int64) ([]repository.LotAssocie, error)
	ListContratsParFournisseur(ctx context.Context, entrepriseID int64) ([]*domain.Contrat, error)
}

// EnrichirExpediteur recherche l'expéditeur d'un e-mail par son adresse et
// rassemble tout son contexte métier (identité, lots, contrats) utile au
// routage.
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
		if pm != nil && pm.EstFournisseur != nil && *pm.EstFournisseur {
			contrats, err := repo.ListContratsParFournisseur(ctx, personne.ID)
			if err != nil {
				return nil, fmt.Errorf("email: recherche des contrats du fournisseur %s (personne id=%d): %w", adresseEmail, personne.ID, err)
			}
			c.Contrats = contrats
		}
	}

	lots, err := repo.ListLotsParPersonne(ctx, personne.ID)
	if err != nil {
		return nil, fmt.Errorf("email: recherche des lots associés à %s (personne id=%d): %w", adresseEmail, personne.ID, err)
	}
	c.Lots = lots

	return c, nil
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
