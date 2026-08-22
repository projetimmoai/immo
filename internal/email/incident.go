package email

import (
	"context"
	"fmt"

	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/service"
)

// traiterIncident traite un e-mail classé comme incident (ex: ascenseur en
// panne, digicode en panne) — un dysfonctionnement technique ponctuel dans
// la copropriété, par opposition à un sinistre (dégât des eaux, incendie...)
// ou des travaux déjà planifiés.
//
// Couvre la création du ticket (deps.Incident, cf. service.IncidentService)
// — la suite du cycle de vie (rapport d'intervention, vérification,
// facture, paiement...) est pilotée par des appels ultérieurs à
// IncidentService, pas depuis ce routeur d'e-mail entrant (cf.
// docs/cycle-vie-incident.md).
//
// LotID reste toujours nil pour l'instant (partie commune) : déterminer
// précisément quel lot est concerné par un incident signalé par e-mail
// n'est pas encore implémenté (cf. service.CreerIncidentInput).
func traiterIncident(ctx context.Context, deps ActionDeps, ctxRoutage domain.ContexteRoutage, _ ResolutionAction, objet, corpsTexte string) error {
	if deps.Incident == nil {
		return fmt.Errorf("email: traitement incident : aucun IncidentService configuré (ActionDeps.Incident)")
	}

	var creePar *int64
	if ctxRoutage.Personne != nil && ctxRoutage.Role != nil && *ctxRoutage.Role == domain.RoleGestionnaire {
		creePar = &ctxRoutage.Personne.ID
	}

	_, _, err := deps.Incident.CreerIncident(ctx, service.CreerIncidentInput{
		SourceID:      ctxRoutage.SourceID,
		CoproprieteID: ctxRoutage.CoproprieteID,
		CreePar:       creePar,
		Objet:         objet,
		CorpsTexte:    corpsTexte,
	})
	if err != nil {
		return fmt.Errorf("email: traitement incident (copropriete_id=%d): %w", ctxRoutage.CoproprieteID, err)
	}
	return nil
}
