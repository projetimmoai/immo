package email

import (
	"context"
	"fmt"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
)

// ResolutionAction est le résultat du routage d'un e-mail dont la
// copropriété a déjà été identifiée : l'action retenue (une des
// descriptions de la table action) et une explication, avec un indice de
// confiance.
type ResolutionAction struct {
	Action    string
	Confiance float64 // entre 0 (aucune confiance) et 1 (certitude)
	Raison    string
}

// actionDecideur est la portion de claudeapi.Client utilisée ici — une
// interface étroite pour pouvoir tester avec un faux (cf. router_test.go).
type actionDecideur interface {
	DecideAction(ctx context.Context, actions []domain.Action, ctxRoutage domain.ContexteRoutage, objet, corpsTexte string) (claudeapi.DecisionAction, error)
}

// gestionnaireAction est le type de fonction appelée pour traiter un
// e-mail une fois l'action décidée — une par action connue (cf.
// internal/email/action_*.go, un fichier par action : chacune va accumuler
// sa propre logique métier, ce qui justifie de les garder séparées plutôt
// que dans un seul gros fichier). Toutes sont encore vides à ce stade :
// seule la décision de la nature de l'action est implémentée pour
// l'instant.
type gestionnaireAction func(ctx context.Context, ctxRoutage domain.ContexteRoutage, decision ResolutionAction, objet, corpsTexte string) error

// gestionnairesAction associe chaque action connue (voir les constantes
// domain.Action*) à la fonction qui la traite.
var gestionnairesAction = map[string]gestionnaireAction{
	domain.ActionSinistre:              traiterSinistre,
	domain.ActionIncident:              traiterIncident,
	domain.ActionAssembleeGenerale:     traiterAssembleeGenerale,
	domain.ActionAutre:                 traiterAutre,
	domain.ActionComptabilite:          traiterComptabilite,
	domain.ActionMutation:              traiterMutation,
	domain.ActionContentieux:           traiterContentieux,
	domain.ActionTravaux:               traiterTravaux,
	domain.ActionControleGestion:       traiterControleGestion,
	domain.ActionDemandeAdministrative: traiterDemandeAdministrative,
}

// RouterEmail détermine l'action à appliquer à un e-mail dont la
// copropriété a déjà été identifiée (cf. DetermineCopropriete), à partir
// du contexte de routage (personne, copropriete, lots/contrats éventuels,
// cf. NouveauContexteRoutage) et du contenu du message, puis appelle la
// fonction de traitement correspondante (cf. gestionnairesAction).
//
// actions est la liste complète des actions possibles (table action),
// à charger une fois par l'appelant (cf. repository.Client.ListActions)
// plutôt qu'à chaque appel.
func RouterEmail(ctx context.Context, claude actionDecideur, actions []*domain.Action, ctxRoutage domain.ContexteRoutage, objet, corpsTexte string) (ResolutionAction, error) {
	if len(actions) == 0 {
		return ResolutionAction{}, fmt.Errorf("email: routage impossible : aucune action disponible (table action vide)")
	}

	actionsValeurs := make([]domain.Action, len(actions))
	for i, a := range actions {
		actionsValeurs[i] = *a
	}

	decision, err := claude.DecideAction(ctx, actionsValeurs, ctxRoutage, objet, corpsTexte)
	if err != nil {
		return ResolutionAction{}, fmt.Errorf("email: détermination de l'action via Claude: %w", err)
	}

	if !actionConnue(actions, decision.Action) {
		// Sécurité : Claude n'est censé choisir que parmi les actions
		// fournies. S'il en invente une, on ne lui fait pas confiance
		// plutôt que de router vers une action qui n'existe pas.
		return ResolutionAction{
			Raison: fmt.Sprintf("réponse Claude incohérente : action %q ne figure pas parmi les %d actions fournies", decision.Action, len(actions)),
		}, nil
	}
	res := ResolutionAction{Action: decision.Action, Confiance: decision.Confiance, Raison: decision.Raison}

	gestionnaire, ok := gestionnairesAction[res.Action]
	if !ok {
		return res, fmt.Errorf("email: aucune fonction de traitement enregistrée pour l'action %q", res.Action)
	}
	if err := gestionnaire(ctx, ctxRoutage, res, objet, corpsTexte); err != nil {
		return res, fmt.Errorf("email: traitement de l'action %q: %w", res.Action, err)
	}
	return res, nil
}

func actionConnue(actions []*domain.Action, description string) bool {
	for _, a := range actions {
		if a.Description == description {
			return true
		}
	}
	return false
}

// NouveauContexteRoutage construit le domain.ContexteRoutage d'un e-mail à
// partir du Contexte enrichi de son expéditeur et de la copropriété
// retenue par DetermineCopropriete. Retourne nil si res.CoproprieteID est
// nil (rien à router : la copropriété n'a pas été identifiée).
func NouveauContexteRoutage(ec *Contexte, res ResolutionCopropriete) *domain.ContexteRoutage {
	if res.CoproprieteID == nil {
		return nil
	}
	cr := &domain.ContexteRoutage{
		Personne:             ec.Personne,
		Role:                 res.Role,
		CoproprieteID:        *res.CoproprieteID,
		CoproprieteReference: res.CoproprieteReference,
	}
	for _, lot := range ec.Lots {
		if lot.CoproprieteID == cr.CoproprieteID {
			cr.LotsReferences = append(cr.LotsReferences, lot.LotReference)
		}
	}
	for _, contrat := range ec.Contrats {
		if contrat.CoproprieteID == cr.CoproprieteID && contrat.NumeroContrat != nil {
			cr.ContratsNumeros = append(cr.ContratsNumeros, *contrat.NumeroContrat)
		}
	}
	return cr
}
