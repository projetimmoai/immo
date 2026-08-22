package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// actionsOccupant liste les actions proposées à Claude pour classifier un
// e-mail dont l'expéditeur a été identifié comme occupant (locataire ou
// propriétaire occupant) d'un lot de la copropriété retenue. Ce n'est pas
// la liste complète de la table action, seulement celles déjà couvertes
// pour ce rôle — à compléter au fur et à mesure (avec, pour chaque
// nouvelle action, sa fonction de traitement dans gestionnairesAction).
var actionsOccupant = []string{
	domain.ActionIncident,
	domain.ActionSinistre,
	domain.ActionTravaux,
	domain.ActionDemandeAdministrative,
	domain.ActionMutation,
}

// RouterOccupant classifie, parmi actionsOccupant, un e-mail dont
// l'expéditeur a été identifié comme occupant (cf. NouveauContexteRoutage),
// puis dispatch chaque demande identifiée vers la fonction de traitement de
// son action — un e-mail peut en contenir plusieurs (cf. routerVersActions).
//
// actions est la liste complète des actions possibles (table action), à
// charger une fois par l'appelant (cf. repository.Client.ListActions)
// plutôt qu'à chaque appel — seul le sous-ensemble pertinent pour un
// occupant (actionsOccupant) est effectivement proposé à Claude.
func RouterOccupant(ctx context.Context, claude actionDecideur, deps ActionDeps, actions []*domain.Action, ctxRoutage domain.ContexteRoutage, objet, corpsTexte string) ([]ResolutionAction, error) {
	return routerVersActions(ctx, claude, deps, filtrerActions(actions, actionsOccupant), ctxRoutage, objet, corpsTexte)
}
