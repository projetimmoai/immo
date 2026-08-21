package email

import (
	"context"

	"github.com/projetimmoai/immo/internal/domain"
)

// traiterSinistre traite un e-mail classé comme sinistre (ex: dégât des
// eaux, incendie) — un évènement grave nécessitant en général une
// déclaration d'assurance, par opposition à un simple incident technique.
//
// TODO: implémenter (création du sinistre en base, notification...).
func traiterSinistre(_ context.Context, _ domain.ContexteRoutage, _ ResolutionAction, _, _ string) error {
	return nil
}
