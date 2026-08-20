package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterDemandeAdministrative traite un e-mail routé vers l'action
// "demande_administrative" (cf. domain.ActionDemandeAdministrative). Pas
// encore implémenté.
func traiterDemandeAdministrative(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	// TODO: implémenter.
	return nil
}
