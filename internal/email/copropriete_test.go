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

type fakeLogRepo struct {
	logTypeID    int64
	logTypeErr   error
	insertErr    error
	insertedLogs []*domain.Log
}

func (f *fakeLogRepo) LogTypeID(_ context.Context, _ string) (int64, error) {
	return f.logTypeID, f.logTypeErr
}

func (f *fakeLogRepo) InsertLog(_ context.Context, l *domain.Log) (*domain.Log, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	f.insertedLogs = append(f.insertedLogs, l)
	return l, nil
}

func TestDetermineCoproprieteExpediteurInconnu(t *testing.T) {
	res, err := DetermineCopropriete(context.Background(), nil, nil, &Contexte{Connu: false}, "Objet", "Corps")
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

	res, err := DetermineCopropriete(context.Background(), nil, nil, ec, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if res.CoproprieteID != nil {
		t.Errorf("CoproprieteID = %v, attendu nil (aucun candidat)", res.CoproprieteID)
	}
}

func TestDetermineCoproprieteUnSeulCandidatCoproprietaire(t *testing.T) {
	cop1 := "COP1"
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleCoproprietaire},
		RolesCopropriete: []repository.PersonneRoleAssociee{
			{Role: "coproprietaire", CoproprieteID: int64Ptr(1), CoproprieteReference: &cop1},
		},
	}

	res, err := DetermineCopropriete(context.Background(), nil, nil, ec, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if res.CoproprieteID == nil || *res.CoproprieteID != 1 || res.CoproprieteReference != "COP1" {
		t.Fatalf("résultat = %+v, attendu copropriete id=1 COP1", res)
	}
	if res.Confiance != 1 {
		t.Errorf("Confiance = %v, attendu 1 (candidat unique)", res.Confiance)
	}
	if res.Role == nil || *res.Role != domain.RoleCoproprietaire {
		t.Errorf("Role = %v, attendu coproprietaire (seul rôle du candidat unique)", res.Role)
	}
}

func TestDetermineCoproprieteUnSeulCandidatMultiRoles(t *testing.T) {
	// Coproprietaire ET occupant de la même (et unique) copropriete : un seul
	// candidat malgré 2 rôles -> toujours facile, mais Role reste
	// indéterminé (ambigu entre les 2 rôles du même candidat).
	cop1 := "COP1"
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleCoproprietaire, domain.RoleOccupant},
		RolesCopropriete: []repository.PersonneRoleAssociee{
			{Role: "coproprietaire", CoproprieteID: int64Ptr(1), CoproprieteReference: &cop1},
			{Role: "occupant", CoproprieteID: int64Ptr(1), CoproprieteReference: &cop1},
		},
	}

	res, err := DetermineCopropriete(context.Background(), nil, nil, ec, "Objet", "Corps")
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

	res, err := DetermineCopropriete(context.Background(), decideur, nil, ec, "Objet", "Corps")
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

	res, err := DetermineCopropriete(context.Background(), decideur, nil, ec, "Objet", "Corps")
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

	res, err := DetermineCopropriete(context.Background(), decideur, nil, ec, "Objet", "Corps")
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

	_, err := DetermineCopropriete(context.Background(), nil, nil, ec, "Objet", "Corps")
	if err == nil {
		t.Fatal("attendu une erreur (plusieurs candidats, aucun client Claude configuré), obtenu nil")
	}
}

func TestDetermineCoproprieteLogueSiConfianceFaible(t *testing.T) {
	gestionnaire := &domain.Personne{ID: 42}
	ec := &Contexte{
		Connu:    true,
		Personne: gestionnaire,
		Roles:    []domain.Role{domain.RoleGestionnaire},
		CoproprietesGestion: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
			{CoproprieteID: 2, CoproprieteReference: "COP2"},
		},
	}
	cop1 := int64(1)
	decideur := &fakeDecideur{decision: claudeapi.DecisionCopropriete{CoproprieteID: &cop1, Confiance: 0.5, Raison: "peu sûr"}}
	logs := &fakeLogRepo{logTypeID: 99}

	res, err := DetermineCopropriete(context.Background(), decideur, logs, ec, "Objet", "Corps")
	if err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if res.CoproprieteID == nil || *res.CoproprieteID != 1 {
		t.Fatalf("résultat = %+v, attendu copropriete id=1 malgré la confiance faible", res)
	}
	if len(logs.insertedLogs) != 1 {
		t.Fatalf("logs insérés = %+v, attendu 1 (confiance 0.5 < seuil)", logs.insertedLogs)
	}
	l := logs.insertedLogs[0]
	if l.LogTypeID != 99 {
		t.Errorf("LogTypeID = %d, attendu 99", l.LogTypeID)
	}
	if l.PersonneID == nil || *l.PersonneID != 42 {
		t.Errorf("PersonneID = %v, attendu 42", l.PersonneID)
	}
	if l.CoproprieteID == nil || *l.CoproprieteID != 1 {
		t.Errorf("CoproprieteID = %v, attendu 1 (copropriete retenue malgré la confiance faible)", l.CoproprieteID)
	}
	if l.Message == nil || *l.Message == "" {
		t.Error("Message vide, attendu une explication")
	}
}

func TestDetermineCoproprieteNeLoguePasSiConfianceSuffisante(t *testing.T) {
	ec := &Contexte{
		Connu:    true,
		Personne: &domain.Personne{ID: 42},
		Roles:    []domain.Role{domain.RoleGestionnaire},
		CoproprietesGestion: []repository.CoproprieteAssociee{
			{CoproprieteID: 1, CoproprieteReference: "COP1"},
			{CoproprieteID: 2, CoproprieteReference: "COP2"},
		},
	}
	cop1 := int64(1)
	decideur := &fakeDecideur{decision: claudeapi.DecisionCopropriete{CoproprieteID: &cop1, Confiance: 0.8, Raison: "sûr"}}
	logs := &fakeLogRepo{logTypeID: 99}

	if _, err := DetermineCopropriete(context.Background(), decideur, logs, ec, "Objet", "Corps"); err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if len(logs.insertedLogs) != 0 {
		t.Fatalf("logs insérés = %+v, attendu aucun (confiance = seuil, pas < seuil)", logs.insertedLogs)
	}
}

func TestDetermineCoproprieteLogueAucunCandidat(t *testing.T) {
	// Expéditeur connu, gestionnaire mais aucune copropriete gérée : aussi
	// un échec d'identification à consigner (confiance 0).
	ec := &Contexte{Connu: true, Personne: &domain.Personne{ID: 7}, Roles: []domain.Role{domain.RoleGestionnaire}}
	logs := &fakeLogRepo{logTypeID: 99}

	if _, err := DetermineCopropriete(context.Background(), nil, logs, ec, "Objet", "Corps"); err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
	if len(logs.insertedLogs) != 1 {
		t.Fatalf("logs insérés = %+v, attendu 1", logs.insertedLogs)
	}
	if logs.insertedLogs[0].CoproprieteID != nil {
		t.Errorf("CoproprieteID = %v, attendu nil (aucun candidat)", logs.insertedLogs[0].CoproprieteID)
	}
}

func TestDetermineCoproprieteRepoLogAbsentNePanicPas(t *testing.T) {
	// repo == nil (pas encore branché) : ne doit jamais paniquer, même
	// quand la confiance est faible et devrait normalement être consignée.
	ec := &Contexte{Connu: true, Personne: &domain.Personne{ID: 7}, Roles: []domain.Role{domain.RoleGestionnaire}}

	if _, err := DetermineCopropriete(context.Background(), nil, nil, ec, "Objet", "Corps"); err != nil {
		t.Fatalf("DetermineCopropriete: %v", err)
	}
}

func TestCandidatsCoproprieteFusionRoles(t *testing.T) {
	// Même copropriete via 2 sources différentes (lots ET gestion) : un
	// seul candidat, avec les 2 rôles cumulés.
	cop1 := "COP1"
	ec := &Contexte{
		Connu: true,
		Roles: []domain.Role{domain.RoleCoproprietaire, domain.RoleGestionnaire},
		RolesCopropriete: []repository.PersonneRoleAssociee{
			{Role: "coproprietaire", CoproprieteID: int64Ptr(1), CoproprieteReference: &cop1},
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
		t.Errorf("Roles = %+v, attendu coproprietaire ET gestionnaire cumulés", candidats[0].Roles)
	}
}
