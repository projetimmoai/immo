package email

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/repository"
)

type fakeDecideur struct {
	decision claudeapi.DecisionCopropriete
	err      error
	appele   bool
	// candidatsRecus est rempli avec les candidats reçus au dernier appel,
	// pour vérifier ce qu'on a effectivement envoyé à Claude.
	candidatsRecus []domain.CandidatCopropriete
}

func (f *fakeDecideur) DecideCopropriete(_ context.Context, candidats []domain.CandidatCopropriete, _, _ string) (claudeapi.DecisionCopropriete, error) {
	f.appele = true
	f.candidatsRecus = candidats
	return f.decision, f.err
}

func TestDetermineCoproprieteExpediteurInconnu(t *testing.T) {
	res, err := DetermineCopropriete(context.Background(), nil, &Contexte{Connu: false}, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if res.CoproprieteID != nil {
		t.Errorf("CoproprieteID = %v, attendu nil", res.CoproprieteID)
	}
	if res.Confiance != 0 {
		t.Errorf("Confiance = %v, attendu 0", res.Confiance)
	}
}

func TestDetermineCoproprieteAucunCandidat(t *testing.T) {
	ec := &Contexte{Connu: true, Roles: []domain.Role{domain.RoleGestionnaire}} // gestionnaire mais aucune copropriete gérée

	res, err := DetermineCopropriete(context.Background(), nil, ec, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if res.CoproprieteID != nil {
		t.Errorf("CoproprieteID = %v, attendu nil (aucun candidat)", res.CoproprieteID)
	}
}

func TestDetermineCoproprieteUnSeulCandidatClient(t *testing.T) {
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleClient},
		Coproprietes: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
		},
	}

	res, err := DetermineCopropriete(context.Background(), nil, ec, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if res.CoproprieteID == nil || *res.CoproprieteID != 1 || res.CoproprieteReference != "COP1" {
		t.Fatalf("résultat = %+v, attendu copropriete id=1 COP1", res)
	}
	if res.Confiance != 1 {
		t.Errorf("Confiance = %v, attendu 1 (candidat unique)", res.Confiance)
	}
	if res.Role == nil || *res.Role != domain.RoleClient {
		t.Errorf("Role = %v, attendu client (seul rôle du candidat unique)", res.Role)
	}
}

func TestDetermineCoproprieteUnSeulCandidatMultiRoles(t *testing.T) {
	// Client ET occupant de la même (et unique) copropriete : un seul
	// candidat malgré 2 rôles -> toujours facile, mais Role reste
	// indéterminé (ambigu entre les 2 rôles du même candidat).
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleClient, domain.RoleOccupant},
		Coproprietes: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
		},
	}

	res, err := DetermineCopropriete(context.Background(), nil, ec, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if res.CoproprieteID == nil || *res.CoproprieteID != 1 {
		t.Fatalf("résultat = %+v, attendu copropriete id=1", res)
	}
	if res.Role != nil {
		t.Errorf("Role = %v, attendu nil (2 rôles pour ce candidat unique)", res.Role)
	}
}

func TestDetermineCopropretePlusieursCandidatsAppelleClaude(t *testing.T) {
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleGestionnaire},
		CoproprietesGestion: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
			{CoproprieteID: 2, CoproprieteReference: "COP2"},
		},
	}
	cop2 := int64(2)
	decideur := &fakeDecideur{decision: claudeapi.DecisionCopropriete{CoproprieteID: &cop2, Confiance: 0.9, Raison: "test"}}

	res, err := DetermineCopropriete(context.Background(), decideur, ec, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if !decideur.appele {
		t.Fatal("attendu un appel à Claude (plusieurs candidats)")
	}
	if len(decideur.candidatsRecus) != 2 {
		t.Fatalf("candidats reçus par Claude = %+v, attendu 2", decideur.candidatsRecus)
	}
	if res.CoproprieteID == nil || *res.CoproprieteID != 2 || res.CoproprieteReference != "COP2" {
		t.Fatalf("résultat = %+v, attendu copropriete id=2 COP2", res)
	}
	if res.Confiance != 0.9 {
		t.Errorf("Confiance = %v, attendu 0.9", res.Confiance)
	}
}

func TestDetermineCoproprieteClaudeIndetermine(t *testing.T) {
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleGestionnaire},
		CoproprietesGestion: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
			{CoproprieteID: 2, CoproprieteReference: "COP2"},
		},
	}
	decideur := &fakeDecideur{decision: claudeapi.DecisionCopropriete{CoproprieteID: nil, Confiance: 0.1, Raison: "pas assez d'info"}}

	res, err := DetermineCopropriete(context.Background(), decideur, ec, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if res.CoproprieteID != nil {
		t.Errorf("CoproprieteID = %v, attendu nil (Claude indéterminé)", res.CoproprieteID)
	}
	if res.Confiance != 0.1 {
		t.Errorf("Confiance = %v, attendu 0.1", res.Confiance)
	}
}

func TestDetermineCoproprieteClaudeReponseIncoherente(t *testing.T) {
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleGestionnaire},
		CoproprietesGestion: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
			{CoproprieteID: 2, CoproprieteReference: "COP2"},
		},
	}
	inventee := int64(999) // n'existe pas parmi les candidats
	decideur := &fakeDecideur{decision: claudeapi.DecisionCopropriete{CoproprieteID: &inventee, Confiance: 0.9, Raison: "hallucination"}}

	res, err := DetermineCopropriete(context.Background(), decideur, ec, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if res.CoproprieteID != nil {
		t.Errorf("CoproprieteID = %v, attendu nil (réponse Claude rejetée, id inconnu)", res.CoproprieteID)
	}
	if res.Confiance != 0 {
		t.Errorf("Confiance = %v, attendu 0 (réponse rejetée)", res.Confiance)
	}
}

func TestDetermineCopropretePlusieursCandidatsSansClaudeConfigure(t *testing.T) {
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleGestionnaire},
		CoproprietesGestion: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
			{CoproprieteID: 2, CoproprieteReference: "COP2"},
		},
	}

	_, err := DetermineCopropriete(context.Background(), nil, ec, "Objet", "Corps")
	if err == nil {
		t.Fatal("attendu une erreur (plusieurs candidats, aucun client Claude configuré), obtenu nil")
	}
}

func TestCandidatsCoproprieteFusionRoles(t *testing.T) {
	// Même copropriete via 2 sources différentes (lots ET gestion) : un
	// seul candidat, avec les 2 rôles cumulés.
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleClient, domain.RoleGestionnaire},
		Coproprietes: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
		},
		CoproprietesGestion: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
		},
	}

	candidats := candidatsCoproprietes(ec)
	if len(candidats) != 1 {
		t.Fatalf("candidats = %+v, attendu 1 (même copropriete via 2 sources)", candidats)
	}
	if len(candidats[0].Roles) != 2 {
		t.Errorf("Roles = %+v, attendu client ET gestionnaire cumulés", candidats[0].Roles)
	}
}
