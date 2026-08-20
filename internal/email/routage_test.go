package email

import (
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/repository"
)

func TestDecideRouteExpediteurInconnu(t *testing.T) {
	d := DecideRoute(&Contexte{Connu: false}, "Bonjour", "un message quelconque")
	if d.Action != domain.ActionIndetermine {
		t.Errorf("Action = %q, attendu %q", d.Action, domain.ActionIndetermine)
	}
}

func TestDecideRouteContexteNil(t *testing.T) {
	d := DecideRoute(nil, "Bonjour", "un message quelconque")
	if d.Action != domain.ActionIndetermine {
		t.Errorf("Action = %q, attendu %q", d.Action, domain.ActionIndetermine)
	}
}

func TestDecideRouteMotsCles(t *testing.T) {
	tests := []struct {
		nom     string
		objet   string
		corps   string
		attendu string
	}{
		{"sinistre", "Déclaration de sinistre", "", domain.ActionSinistre},
		{"incident accent", "Fuite d'eau", "", domain.ActionIncident},
		{"ascenseur", "", "L'ascenseur est en panne", domain.ActionIncident},
		{"assemblee generale", "Convocation à l'assemblée générale", "", domain.ActionAssembleeGenerale},
	}

	connu := &Contexte{Connu: true, Personne: &domain.Personne{ID: 1}}
	for _, tt := range tests {
		t.Run(tt.nom, func(t *testing.T) {
			d := DecideRoute(connu, tt.objet, tt.corps)
			if d.Action != tt.attendu {
				t.Errorf("Action = %q, attendu %q (raison: %s)", d.Action, tt.attendu, d.Raison)
			}
		})
	}
}

func TestDecideRouteFournisseurSansMotCle(t *testing.T) {
	vrai := true
	ctx := &Contexte{
		Connu:          true,
		Personne:       &domain.Personne{ID: 2},
		PersonneMorale: &domain.PersonneMorale{ID: 20, EstFournisseur: &vrai},
		Roles:          []domain.Role{domain.RoleFournisseur},
		Contrats:       []repository.ContratAssocie{{ContratID: 200}},
	}
	d := DecideRoute(ctx, "Bonjour", "Merci de votre confiance")
	if d.Action != domain.ActionAutre {
		t.Errorf("Action = %q, attendu %q", d.Action, domain.ActionAutre)
	}
}

func TestDecideRouteConnuSansMotCleNiFournisseur(t *testing.T) {
	ctx := &Contexte{Connu: true, Personne: &domain.Personne{ID: 3}}
	d := DecideRoute(ctx, "Bonjour", "Comment allez-vous ?")
	if d.Action != domain.ActionAutre {
		t.Errorf("Action = %q, attendu %q", d.Action, domain.ActionAutre)
	}
}
