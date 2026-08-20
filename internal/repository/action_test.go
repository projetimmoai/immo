package repository

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
)

func TestListActions(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	actions, err := c.ListActions(ctx)
	if err != nil {
		t.Fatalf("ListActions: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("ListActions: attendu au moins une action, obtenu aucune")
	}

	var trouveSinistre bool
	for _, a := range actions {
		if a.Description == domain.ActionSinistre {
			trouveSinistre = true
		}
	}
	if !trouveSinistre {
		t.Errorf("ListActions: attendu de trouver %q parmi %+v", domain.ActionSinistre, actions)
	}
}
