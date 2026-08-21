package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterDemandeAdministrative traite un e-mail classé comme demande
// administrative (ex: PVs d'AG, règlement de copropriété, diagnostics) —
// une demande de document lié à la copropriété.
//
// TODO: implémenter.
func traiterDemandeAdministrative(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	return nil
}
