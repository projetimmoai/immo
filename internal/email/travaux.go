package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterTravaux traite un e-mail classé comme travaux (ex: travail mal
// fait, délai d'intervention, comportement de l'entreprise) — une remarque
// à propos de travaux en cours ou déjà réalisés dans la copropriété.
//
// TODO: implémenter.
func traiterTravaux(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	return nil
}
