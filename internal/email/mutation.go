package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterMutation traite un e-mail classé comme mutation (ex: départ
// locataire, changement de numéro de téléphone) — un changement à
// répercuter sur les informations connues de l'occupant ou de son lot.
//
// TODO: implémenter.
func traiterMutation(_ context.Context, _ ActionDeps, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	return nil
}
