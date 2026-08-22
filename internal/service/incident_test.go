package service

import (
	"context"
	"testing"
	"time"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/repository"
)

// Identifiants fixes utilisés par fakeIncidentRepo — un ID par description
// connue, pour que les fonctions de résolution (cf. TicketStatutID...)
// soient déterministes dans les tests.
const (
	idActionIncident              = 1
	idStatutNouveau               = 10
	idStatutEnAttenteEmetteur     = 11
	idStatutEnAttenteGestionnaire = 12
	idStatutEnAttenteTiers        = 13
	idStatutResolu                = 14
	idCategorieFuiteEau           = 20
	idUrgenceFaible               = 30
	idUrgenceMoyen                = 31
	idModeConfirmationOccupant    = 40
	idModeJugeeInutile            = 41
	idResultatPositive            = 50
	idResultatNegative            = 51
	idFactureRecue                = 60
	idFactureValidee              = 61
	idFacturePayee                = 62
	idFactureRapprochee           = 63
)

// fakeIncidentRepo simule incidentRepo — un registre en mémoire plutôt qu'un
// vrai accès base, pour tester IncidentService indépendamment de
// repository.Client (cf. router_test.go/contexte_test.go pour le même
// principe ailleurs dans le projet).
type fakeIncidentRepo struct {
	categories   []domain.CategorieTechnique
	urgences     []domain.NiveauUrgence
	copropriete  *domain.Copropriete
	contratActif *repository.ContratActif

	prochainTicketID   int64
	tickets            map[int64]*domain.Ticket
	incidents          map[int64]*domain.Incident
	factures           map[int64]*domain.Facture
	prochaineFactureID int64

	// Enregistrements des appels, pour les assertions des tests.
	statutsAppliques    map[int64]int64 // ticket_id -> dernier statut_id appliqué
	prestatairesDefinis map[int64]int64 // ticket_id -> prestataire_id
}

func newFakeIncidentRepo() *fakeIncidentRepo {
	return &fakeIncidentRepo{
		prochainTicketID:    1,
		prochaineFactureID:  1,
		tickets:             map[int64]*domain.Ticket{},
		incidents:           map[int64]*domain.Incident{},
		factures:            map[int64]*domain.Facture{},
		statutsAppliques:    map[int64]int64{},
		prestatairesDefinis: map[int64]int64{},
	}
}

func (f *fakeIncidentRepo) ActionID(_ context.Context, description string) (int64, error) {
	if description == domain.ActionIncident {
		return idActionIncident, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) TicketStatutID(_ context.Context, description string) (int64, error) {
	switch description {
	case domain.TicketStatutNouveau:
		return idStatutNouveau, nil
	case domain.TicketStatutEnAttenteEmetteur:
		return idStatutEnAttenteEmetteur, nil
	case domain.TicketStatutEnAttenteGestionnaire:
		return idStatutEnAttenteGestionnaire, nil
	case domain.TicketStatutEnAttenteTiers:
		return idStatutEnAttenteTiers, nil
	case domain.TicketStatutResolu:
		return idStatutResolu, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) CategorieTechniqueID(_ context.Context, description string) (int64, error) {
	if description == "fuite_eau" {
		return idCategorieFuiteEau, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) NiveauUrgenceID(_ context.Context, description string) (int64, error) {
	switch description {
	case domain.NiveauUrgenceFaible:
		return idUrgenceFaible, nil
	case domain.NiveauUrgenceMoyen:
		return idUrgenceMoyen, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) ListCategorieTechnique(_ context.Context) ([]domain.CategorieTechnique, error) {
	return f.categories, nil
}

func (f *fakeIncidentRepo) ListNiveauUrgence(_ context.Context) ([]domain.NiveauUrgence, error) {
	return f.urgences, nil
}

func (f *fakeIncidentRepo) InsertIncident(_ context.Context, in repository.CreerIncidentInput) (*domain.Ticket, *domain.Incident, error) {
	id := f.prochainTicketID
	f.prochainTicketID++
	ticket := &domain.Ticket{ID: id, Reference: "TIC1", ActionID: in.ActionID, StatutID: in.StatutID, SourceID: in.SourceID, CoproprieteID: in.CoproprieteID}
	incident := &domain.Incident{TicketID: id, CategorieTechniqueID: in.CategorieTechniqueID, UrgenceID: in.UrgenceID, MontantEstimeCentimes: in.MontantEstimeCentimes}
	f.tickets[id] = ticket
	f.incidents[id] = incident
	f.statutsAppliques[id] = in.StatutID
	return ticket, incident, nil
}

func (f *fakeIncidentRepo) FindCoproprieteByID(_ context.Context, _ int64) (*domain.Copropriete, error) {
	return f.copropriete, nil
}

func (f *fakeIncidentRepo) FindContratActif(_ context.Context, _, _ int64) (*repository.ContratActif, error) {
	return f.contratActif, nil
}

func (f *fakeIncidentRepo) SetIncidentPrestataire(_ context.Context, ticketID, prestataireID int64) error {
	f.prestatairesDefinis[ticketID] = prestataireID
	if inc, ok := f.incidents[ticketID]; ok {
		inc.PrestataireID = &prestataireID
	}
	return nil
}

func (f *fakeIncidentRepo) UpdateTicketStatut(_ context.Context, ticketID, statutID int64) error {
	f.statutsAppliques[ticketID] = statutID
	if t, ok := f.tickets[ticketID]; ok {
		t.StatutID = statutID
	}
	return nil
}

func (f *fakeIncidentRepo) SetIncidentRapportIntervention(_ context.Context, ticketID int64, rapport string) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.RapportIntervention = &rapport
	}
	return nil
}

func (f *fakeIncidentRepo) ModeVerificationID(_ context.Context, description string) (int64, error) {
	switch description {
	case domain.ModeVerificationConfirmationOccupant:
		return idModeConfirmationOccupant, nil
	case domain.ModeVerificationJugeeInutile:
		return idModeJugeeInutile, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) VerificationResultatID(_ context.Context, description string) (int64, error) {
	switch description {
	case domain.VerificationResultatPositive:
		return idResultatPositive, nil
	case domain.VerificationResultatNegative:
		return idResultatNegative, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) SetIncidentModeVerification(_ context.Context, ticketID, modeID int64, resultatID *int64) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.ModeVerificationID = &modeID
		inc.VerificationResultatID = resultatID
	}
	return nil
}

func (f *fakeIncidentRepo) SetIncidentVerificationResultat(_ context.Context, ticketID, resultatID int64, dateResolution *time.Time) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.VerificationResultatID = &resultatID
		inc.DateResolution = dateResolution
	}
	return nil
}

func (f *fakeIncidentRepo) FindIncidentByTicketID(_ context.Context, ticketID int64) (*domain.Incident, error) {
	return f.incidents[ticketID], nil
}

func (f *fakeIncidentRepo) FactureStatutID(_ context.Context, description string) (int64, error) {
	switch description {
	case domain.FactureStatutRecue:
		return idFactureRecue, nil
	case domain.FactureStatutValidee:
		return idFactureValidee, nil
	case domain.FactureStatutPayee:
		return idFacturePayee, nil
	case domain.FactureStatutRapprochee:
		return idFactureRapprochee, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) InsertFacture(_ context.Context, fa *domain.Facture) (*domain.Facture, error) {
	id := f.prochaineFactureID
	f.prochaineFactureID++
	copie := *fa
	copie.ID = id
	f.factures[fa.TicketID] = &copie
	return &copie, nil
}

func (f *fakeIncidentRepo) ValiderFacture(_ context.Context, factureID, statutValideeID, validePar int64, dateValidation time.Time) error {
	for _, fa := range f.factures {
		if fa.ID == factureID {
			fa.StatutID = statutValideeID
			fa.DateValidation = &dateValidation
			fa.ValidePar = &validePar
		}
	}
	return nil
}

func (f *fakeIncidentRepo) MettreFactureEnPaiement(_ context.Context, factureID, statutPayeeID, payePar int64, datePaiement time.Time) error {
	for _, fa := range f.factures {
		if fa.ID == factureID {
			fa.StatutID = statutPayeeID
			fa.DatePaiement = &datePaiement
			fa.PayePar = &payePar
		}
	}
	return nil
}

func (f *fakeIncidentRepo) RapprocherFacture(_ context.Context, factureID, statutRapprocheeID, rapprochePar int64, dateRapprochement time.Time) error {
	for _, fa := range f.factures {
		if fa.ID == factureID {
			fa.StatutID = statutRapprocheeID
			fa.DateRapprochement = &dateRapprochement
			fa.RapprochePar = &rapprochePar
		}
	}
	return nil
}

func (f *fakeIncidentRepo) FindFactureByTicketID(_ context.Context, ticketID int64) (*domain.Facture, error) {
	return f.factures[ticketID], nil
}

func errIntrouvable(description string) error {
	return &erreurLookup{description: description}
}

type erreurLookup struct{ description string }

func (e *erreurLookup) Error() string { return "lookup introuvable: " + e.description }

// fakeIncidentQualifieur simule incidentQualifieur : renvoie toujours la
// même QualificationIncident, fixée par le test.
type fakeIncidentQualifieur struct {
	qualif claudeapi.QualificationIncident
	err    error
}

func (f *fakeIncidentQualifieur) QualifierIncident(_ context.Context, _ []domain.CategorieTechnique, _ []domain.NiveauUrgence, _, _ string) (claudeapi.QualificationIncident, error) {
	return f.qualif, f.err
}

func categorieFuiteEau() string { return "fuite_eau" }

func TestCreerIncidentSousPlafondAvecContratActif(t *testing.T) {
	repo := newFakeIncidentRepo()
	categorie := categorieFuiteEau()
	repo.contratActif = &repository.ContratActif{ContratID: 900, EntrepriseID: 42}
	plafond := int64(50000)
	repo.copropriete = &domain.Copropriete{ID: 1, PlafondOrdreServiceCentimes: &plafond}
	montant := int64(20000)
	claude := &fakeIncidentQualifieur{qualif: claudeapi.QualificationIncident{
		CategorieTechnique:    &categorie,
		Urgence:               domain.NiveauUrgenceMoyen,
		MontantEstimeCentimes: &montant,
		Confiance:             0.9,
	}}
	svc := &IncidentService{Repo: repo, Claude: claude}

	ticket, incident, err := svc.CreerIncident(context.Background(), CreerIncidentInput{
		SourceID: 5, CoproprieteID: 1, Objet: "Fuite", CorpsTexte: "Il y a une fuite dans la cave",
	})
	if err != nil {
		t.Fatalf("CreerIncident: %v", err)
	}
	if repo.statutsAppliques[ticket.ID] != idStatutEnAttenteTiers {
		t.Errorf("statut = %d, attendu en_attente_tiers (%d) : sous le plafond D, prestataire trouvé", repo.statutsAppliques[ticket.ID], idStatutEnAttenteTiers)
	}
	if repo.prestatairesDefinis[ticket.ID] != 42 {
		t.Errorf("prestataire = %d, attendu 42", repo.prestatairesDefinis[ticket.ID])
	}
	if incident.CategorieTechniqueID == nil || *incident.CategorieTechniqueID != idCategorieFuiteEau {
		t.Errorf("CategorieTechniqueID = %v, attendu %d", incident.CategorieTechniqueID, idCategorieFuiteEau)
	}
}

func TestCreerIncidentSansContratActifAttenteGestionnaire(t *testing.T) {
	repo := newFakeIncidentRepo()
	categorie := categorieFuiteEau()
	repo.contratActif = nil // aucun contrat actif
	claude := &fakeIncidentQualifieur{qualif: claudeapi.QualificationIncident{
		CategorieTechnique: &categorie,
		Urgence:            domain.NiveauUrgenceMoyen,
	}}
	svc := &IncidentService{Repo: repo, Claude: claude}

	ticket, _, err := svc.CreerIncident(context.Background(), CreerIncidentInput{SourceID: 5, CoproprieteID: 1, Objet: "x", CorpsTexte: "y"})
	if err != nil {
		t.Fatalf("CreerIncident: %v", err)
	}
	if repo.statutsAppliques[ticket.ID] != idStatutEnAttenteGestionnaire {
		t.Errorf("statut = %d, attendu en_attente_gestionnaire (%d) : aucun contrat actif", repo.statutsAppliques[ticket.ID], idStatutEnAttenteGestionnaire)
	}
}

func TestCreerIncidentMontantAuDelaDuPlafondAttenteGestionnaire(t *testing.T) {
	repo := newFakeIncidentRepo()
	categorie := categorieFuiteEau()
	repo.contratActif = &repository.ContratActif{ContratID: 900, EntrepriseID: 42}
	plafond := int64(10000)
	repo.copropriete = &domain.Copropriete{ID: 1, PlafondOrdreServiceCentimes: &plafond}
	montant := int64(99999) // dépasse le plafond D
	claude := &fakeIncidentQualifieur{qualif: claudeapi.QualificationIncident{
		CategorieTechnique:    &categorie,
		Urgence:               domain.NiveauUrgenceMoyen,
		MontantEstimeCentimes: &montant,
	}}
	svc := &IncidentService{Repo: repo, Claude: claude}

	ticket, _, err := svc.CreerIncident(context.Background(), CreerIncidentInput{SourceID: 5, CoproprieteID: 1, Objet: "x", CorpsTexte: "y"})
	if err != nil {
		t.Fatalf("CreerIncident: %v", err)
	}
	if repo.statutsAppliques[ticket.ID] != idStatutEnAttenteGestionnaire {
		t.Errorf("statut = %d, attendu en_attente_gestionnaire (%d) : montant au-delà du plafond D, devis requis (pas implémenté)", repo.statutsAppliques[ticket.ID], idStatutEnAttenteGestionnaire)
	}
}

func TestEnregistrerRapportInterventionUrgenceFaibleJugeeInutile(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.incidents[1] = &domain.Incident{TicketID: 1, UrgenceID: int64Ptr(idUrgenceFaible)}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerRapportIntervention(context.Background(), 1, "Fuite réparée"); err != nil {
		t.Fatalf("EnregistrerRapportIntervention: %v", err)
	}
	inc := repo.incidents[1]
	if inc.ModeVerificationID == nil || *inc.ModeVerificationID != idModeJugeeInutile {
		t.Errorf("ModeVerificationID = %v, attendu jugee_inutile (%d)", inc.ModeVerificationID, idModeJugeeInutile)
	}
	if inc.VerificationResultatID == nil || *inc.VerificationResultatID != idResultatPositive {
		t.Errorf("VerificationResultatID = %v, attendu positive (%d)", inc.VerificationResultatID, idResultatPositive)
	}
	if inc.DateResolution == nil {
		t.Error("DateResolution non renseignée alors que la vérification est jugée inutile (positive)")
	}
}

func TestEnregistrerRapportInterventionUrgenceMoyenneDemandeConfirmation(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.incidents[1] = &domain.Incident{TicketID: 1, UrgenceID: int64Ptr(idUrgenceMoyen)}
	repo.tickets[1] = &domain.Ticket{ID: 1}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerRapportIntervention(context.Background(), 1, "Ascenseur redémarré"); err != nil {
		t.Fatalf("EnregistrerRapportIntervention: %v", err)
	}
	inc := repo.incidents[1]
	if inc.ModeVerificationID == nil || *inc.ModeVerificationID != idModeConfirmationOccupant {
		t.Errorf("ModeVerificationID = %v, attendu confirmation_occupant (%d)", inc.ModeVerificationID, idModeConfirmationOccupant)
	}
	if inc.VerificationResultatID != nil {
		t.Errorf("VerificationResultatID = %v, attendu nil (résultat pas encore connu)", inc.VerificationResultatID)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteEmetteur {
		t.Errorf("statut = %d, attendu en_attente_emetteur (%d)", repo.statutsAppliques[1], idStatutEnAttenteEmetteur)
	}
}

func TestEnregistrerConfirmationOccupantNegativeAttenteGestionnaire(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.incidents[1] = &domain.Incident{TicketID: 1}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerConfirmationOccupant(context.Background(), 1, false); err != nil {
		t.Fatalf("EnregistrerConfirmationOccupant: %v", err)
	}
	inc := repo.incidents[1]
	if inc.VerificationResultatID == nil || *inc.VerificationResultatID != idResultatNegative {
		t.Errorf("VerificationResultatID = %v, attendu negative (%d)", inc.VerificationResultatID, idResultatNegative)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteGestionnaire {
		t.Errorf("statut = %d, attendu en_attente_gestionnaire (%d)", repo.statutsAppliques[1], idStatutEnAttenteGestionnaire)
	}
}

func TestMettreEnPaiementRefuseSiVerificationNonPositive(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.incidents[1] = &domain.Incident{TicketID: 1, VerificationResultatID: int64Ptr(idResultatNegative)}
	repo.factures[1] = &domain.Facture{ID: 7, TicketID: 1, StatutID: idFactureValidee}
	svc := &IncidentService{Repo: repo}

	if err := svc.MettreEnPaiement(context.Background(), 1, 99); err == nil {
		t.Fatal("MettreEnPaiement: attendu une erreur (vérification non positive), obtenu nil")
	}
	if repo.factures[1].StatutID != idFactureValidee {
		t.Errorf("StatutID = %d, ne devrait pas avoir changé (paiement refusé)", repo.factures[1].StatutID)
	}
}

func TestMettreEnPaiementRefuseSiFactureNonValidee(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.incidents[1] = &domain.Incident{TicketID: 1, VerificationResultatID: int64Ptr(idResultatPositive)}
	repo.factures[1] = &domain.Facture{ID: 7, TicketID: 1, StatutID: idFactureRecue} // pas encore validée
	svc := &IncidentService{Repo: repo}

	if err := svc.MettreEnPaiement(context.Background(), 1, 99); err == nil {
		t.Fatal("MettreEnPaiement: attendu une erreur (facture non validée), obtenu nil")
	}
}

func TestMettreEnPaiementReussitSiFactureValideeEtVerificationPositive(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.incidents[1] = &domain.Incident{TicketID: 1, VerificationResultatID: int64Ptr(idResultatPositive)}
	repo.factures[1] = &domain.Facture{ID: 7, TicketID: 1, StatutID: idFactureValidee}
	svc := &IncidentService{Repo: repo}

	if err := svc.MettreEnPaiement(context.Background(), 1, 99); err != nil {
		t.Fatalf("MettreEnPaiement: %v", err)
	}
	if repo.factures[1].StatutID != idFacturePayee {
		t.Errorf("StatutID = %d, attendu payee (%d)", repo.factures[1].StatutID, idFacturePayee)
	}
	if repo.factures[1].PayePar == nil || *repo.factures[1].PayePar != 99 {
		t.Errorf("PayePar = %v, attendu 99", repo.factures[1].PayePar)
	}
}

func TestRapprocherFactureClotureLeTicket(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.tickets[1] = &domain.Ticket{ID: 1}
	repo.factures[1] = &domain.Facture{ID: 7, TicketID: 1, StatutID: idFacturePayee}
	svc := &IncidentService{Repo: repo}

	if err := svc.RapprocherFacture(context.Background(), 1, 7, 99); err != nil {
		t.Fatalf("RapprocherFacture: %v", err)
	}
	if repo.factures[1].StatutID != idFactureRapprochee {
		t.Errorf("StatutID = %d, attendu rapprochee (%d)", repo.factures[1].StatutID, idFactureRapprochee)
	}
	if repo.statutsAppliques[1] != idStatutResolu {
		t.Errorf("statut ticket = %d, attendu résolu (%d)", repo.statutsAppliques[1], idStatutResolu)
	}
}

func int64Ptr(v int64) *int64 { return &v }
