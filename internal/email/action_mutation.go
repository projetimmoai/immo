package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterMutation traite un e-mail routé vers l'action "mutation" (cf.
// domain.ActionMutation — changement de propriétaire ou d'occupant d'un
// lot). Pas encore implémenté.
func traiterMutation(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	// TODO: implémenter.
	return nil
}
