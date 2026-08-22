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
	idActionIncident                   = 1
	idStatutNouveau                    = 10
	idStatutEnAttenteEmetteur          = 11
	idStatutEnAttenteGestionnaire      = 12
	idStatutEnAttenteTiers             = 13
	idStatutResolu                     = 14
	idStatutEnAttenteConseilSyndical   = 15
	idStatutEnAttenteAssembleeGenerale = 16
	idStatutLitige                     = 17
	idCategorieFuiteEau                = 20
	idUrgenceFaible                    = 30
	idUrgenceMoyen                     = 31
	idModeConfirmationOccupant         = 40
	idModeJugeeInutile                 = 41
	idModeVerificationGH               = 42
	idResultatPositive                 = 50
	idResultatNegative                 = 51
	idFactureRecue                     = 60
	idFactureValidee                   = 61
	idFacturePayee                     = 62
	idFactureRapprochee                = 63
	idDevisEnAttente                   = 70
	idDevisRecu                        = 71
	idDevisRetenu                      = 72
	idDevisRejete                      = 73
	idAGApprouve                       = 80
	idAGRejete                         = 81
	idReclamationEnvoyee               = 90
	idReclamationAcceptee              = 91
	idReclamationRefusee               = 92
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
	delegation   *domain.ConseilSyndicalDelegation // simplifié : actif dès qu'il couvre le montant, pas de vraies dates

	prochainTicketID       int64
	tickets                map[int64]*domain.Ticket
	incidents              map[int64]*domain.Incident
	factures               map[int64]*domain.Facture
	prochaineFactureID     int64
	devis                  map[int64]*domain.Devis
	prochainDevisID        int64
	reclamations           map[int64]*domain.Reclamation
	prochaineReclamationID int64

	// Enregistrements des appels, pour les assertions des tests.
	statutsAppliques    map[int64]int64 // ticket_id -> dernier statut_id appliqué
	prestatairesDefinis map[int64]int64 // ticket_id -> prestataire_id
}

func newFakeIncidentRepo() *fakeIncidentRepo {
	return &fakeIncidentRepo{
		prochainTicketID:       1,
		prochaineFactureID:     1,
		prochainDevisID:        1,
		tickets:                map[int64]*domain.Ticket{},
		incidents:              map[int64]*domain.Incident{},
		factures:               map[int64]*domain.Facture{},
		devis:                  map[int64]*domain.Devis{},
		prochaineReclamationID: 1,
		reclamations:           map[int64]*domain.Reclamation{},
		statutsAppliques:       map[int64]int64{},
		prestatairesDefinis:    map[int64]int64{},
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
	case domain.TicketStatutEnAttenteConseilSyndical:
		return idStatutEnAttenteConseilSyndical, nil
	case domain.TicketStatutEnAttenteAssembleeGenerale:
		return idStatutEnAttenteAssembleeGenerale, nil
	case domain.TicketStatutLitige:
		return idStatutLitige, nil
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
	case domain.ModeVerificationGH:
		return idModeVerificationGH, nil
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

func (f *fakeIncidentRepo) FindTicketByID(_ context.Context, id int64) (*domain.Ticket, error) {
	return f.tickets[id], nil
}

func (f *fakeIncidentRepo) DevisStatutID(_ context.Context, description string) (int64, error) {
	switch description {
	case domain.DevisStatutEnAttente:
		return idDevisEnAttente, nil
	case domain.DevisStatutRecu:
		return idDevisRecu, nil
	case domain.DevisStatutRetenu:
		return idDevisRetenu, nil
	case domain.DevisStatutRejete:
		return idDevisRejete, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) InsertDevis(_ context.Context, d *domain.Devis) (*domain.Devis, error) {
	id := f.prochainDevisID
	f.prochainDevisID++
	copie := *d
	copie.ID = id
	f.devis[id] = &copie
	return &copie, nil
}

func (f *fakeIncidentRepo) EnregistrerReceptionDevis(_ context.Context, devisID, statutRecuID, montantCentimes int64, dateReception time.Time) error {
	d, ok := f.devis[devisID]
	if !ok {
		return errIntrouvable("devis")
	}
	d.StatutID = statutRecuID
	d.MontantCentimes = &montantCentimes
	d.DateReception = &dateReception
	return nil
}

func (f *fakeIncidentRepo) MarquerDevisStatut(_ context.Context, devisID, statutID int64) error {
	d, ok := f.devis[devisID]
	if !ok {
		return errIntrouvable("devis")
	}
	d.StatutID = statutID
	return nil
}

func (f *fakeIncidentRepo) ListDevisByTicket(_ context.Context, ticketID int64) ([]*domain.Devis, error) {
	var res []*domain.Devis
	for _, d := range f.devis {
		if d.TicketID == ticketID {
			res = append(res, d)
		}
	}
	return res, nil
}

func (f *fakeIncidentRepo) FindDelegationActive(_ context.Context, _, montantCentimes int64) (*domain.ConseilSyndicalDelegation, error) {
	if f.delegation != nil && montantCentimes <= f.delegation.PlafondCentimes {
		return f.delegation, nil
	}
	return nil, nil
}

func (f *fakeIncidentRepo) SetIncidentDevisRetenu(_ context.Context, ticketID, devisID int64) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.DevisRetenuID = &devisID
	}
	return nil
}

func (f *fakeIncidentRepo) SetIncidentAvisCSDemande(_ context.Context, ticketID int64, demandeLe time.Time) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.AvisCSDemandeLe = &demandeLe
	}
	return nil
}

func (f *fakeIncidentRepo) SetIncidentAvisCSRecu(_ context.Context, ticketID int64, recuLe time.Time, texte string) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.AvisCSRecuLe = &recuLe
		inc.AvisCSTexte = &texte
	}
	return nil
}

func (f *fakeIncidentRepo) SetIncidentDecisionCSDemande(_ context.Context, ticketID int64, demandeLe time.Time) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.DecisionCSDemandeLe = &demandeLe
	}
	return nil
}

func (f *fakeIncidentRepo) SetIncidentDecisionCSRecue(_ context.Context, ticketID int64, recueLe time.Time) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.DecisionCSRecueLe = &recueLe
	}
	return nil
}

func (f *fakeIncidentRepo) SetIncidentAGResolution(_ context.Context, ticketID int64, texte string, inscriteLe time.Time) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.AGResolutionTexte = &texte
		inc.AGInscriteLe = &inscriteLe
	}
	return nil
}

func (f *fakeIncidentRepo) SetIncidentAGResultat(_ context.Context, ticketID, resultatID int64, voteeLe time.Time) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.AGResultatID = &resultatID
		inc.AGVoteeLe = &voteeLe
	}
	return nil
}

func (f *fakeIncidentRepo) AGResultatID(_ context.Context, description string) (int64, error) {
	switch description {
	case domain.AGResultatApprouve:
		return idAGApprouve, nil
	case domain.AGResultatRejete:
		return idAGRejete, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) SetIncidentModeVerificationNouveauCycle(_ context.Context, ticketID, modeID int64) error {
	if inc, ok := f.incidents[ticketID]; ok {
		inc.ModeVerificationID = &modeID
		inc.VerificationResultatID = nil
		inc.DateResolution = nil
	}
	return nil
}

func (f *fakeIncidentRepo) ReclamationStatutID(_ context.Context, description string) (int64, error) {
	switch description {
	case domain.ReclamationStatutEnvoyee:
		return idReclamationEnvoyee, nil
	case domain.ReclamationStatutAcceptee:
		return idReclamationAcceptee, nil
	case domain.ReclamationStatutRefusee:
		return idReclamationRefusee, nil
	}
	return 0, errIntrouvable(description)
}

func (f *fakeIncidentRepo) InsertReclamation(_ context.Context, r *domain.Reclamation) (*domain.Reclamation, error) {
	id := f.prochaineReclamationID
	f.prochaineReclamationID++
	copie := *r
	copie.ID = id
	f.reclamations[id] = &copie
	return &copie, nil
}

func (f *fakeIncidentRepo) EnregistrerReponseReclamation(_ context.Context, reclamationID, statutID int64, dateReponse time.Time, reponseTexte string) error {
	r, ok := f.reclamations[reclamationID]
	if !ok {
		return errIntrouvable("reclamation")
	}
	r.StatutID = statutID
	r.DateReponse = &dateReponse
	r.ReponseTexte = &reponseTexte
	return nil
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

// Le cas "montant au-delà du plafond D" est désormais couvert par
// TestCreerIncidentAuDelaDuPlafondDCreeUnDevis (demande de devis) et
// TestCreerIncidentSeuilBFranchiFallbackHumain (mise en concurrence requise,
// fallback humain) — l'ancien comportement "toujours en attente
// gestionnaire au-delà du plafond D" a été remplacé par la chaîne de
// décision réelle (cf. IncidentService.demarrerDevis).

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

func TestEnregistrerConfirmationOccupantNegativeSansPrestataireAttenteGestionnaire(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.incidents[1] = &domain.Incident{TicketID: 1} // pas de prestataire connu (ne devrait pas arriver à ce stade)
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerConfirmationOccupant(context.Background(), 1, false); err != nil {
		t.Fatalf("EnregistrerConfirmationOccupant: %v", err)
	}
	inc := repo.incidents[1]
	if inc.VerificationResultatID == nil || *inc.VerificationResultatID != idResultatNegative {
		t.Errorf("VerificationResultatID = %v, attendu negative (%d)", inc.VerificationResultatID, idResultatNegative)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteGestionnaire {
		t.Errorf("statut = %d, attendu en_attente_gestionnaire (%d) : pas de prestataire, la réclamation est impossible", repo.statutsAppliques[1], idStatutEnAttenteGestionnaire)
	}
}

func TestEnregistrerConfirmationOccupantNegativeDemarreReclamation(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.tickets[1] = &domain.Ticket{ID: 1}
	repo.incidents[1] = &domain.Incident{TicketID: 1, PrestataireID: int64Ptr(42)}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerConfirmationOccupant(context.Background(), 1, false); err != nil {
		t.Fatalf("EnregistrerConfirmationOccupant: %v", err)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteTiers {
		t.Errorf("statut = %d, attendu en_attente_tiers (%d) : réclamation envoyée, en attente de la réponse du prestataire", repo.statutsAppliques[1], idStatutEnAttenteTiers)
	}
	var trouvee *domain.Reclamation
	for _, r := range repo.reclamations {
		if r.TicketID == 1 {
			trouvee = r
		}
	}
	if trouvee == nil {
		t.Fatal("aucune réclamation créée")
	}
	if trouvee.PrestataireID != 42 {
		t.Errorf("PrestataireID = %d, attendu 42", trouvee.PrestataireID)
	}
	if trouvee.StatutID != idReclamationEnvoyee {
		t.Errorf("StatutID = %d, attendu envoyee (%d)", trouvee.StatutID, idReclamationEnvoyee)
	}
}

func TestEnregistrerReponseReclamationAccepteeRetourPhase4(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.tickets[1] = &domain.Ticket{ID: 1}
	repo.reclamations[9] = &domain.Reclamation{ID: 9, TicketID: 1, StatutID: idReclamationEnvoyee}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerReponseReclamation(context.Background(), 1, 9, true, "D'accord, je reviens corriger."); err != nil {
		t.Fatalf("EnregistrerReponseReclamation: %v", err)
	}
	if repo.reclamations[9].StatutID != idReclamationAcceptee {
		t.Errorf("StatutID = %d, attendu acceptee (%d)", repo.reclamations[9].StatutID, idReclamationAcceptee)
	}
	if repo.reclamations[9].ReponseTexte == nil || *repo.reclamations[9].ReponseTexte != "D'accord, je reviens corriger." {
		t.Errorf("ReponseTexte = %v, attendu le texte fourni", repo.reclamations[9].ReponseTexte)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteTiers {
		t.Errorf("statut = %d, attendu en_attente_tiers (%d) : retour phase 4, nouvelle intervention", repo.statutsAppliques[1], idStatutEnAttenteTiers)
	}
}

func TestEnregistrerReponseReclamationRefuseeLitige(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.tickets[1] = &domain.Ticket{ID: 1}
	repo.reclamations[9] = &domain.Reclamation{ID: 9, TicketID: 1, StatutID: idReclamationEnvoyee}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerReponseReclamation(context.Background(), 1, 9, false, "Le travail a été fait correctement."); err != nil {
		t.Fatalf("EnregistrerReponseReclamation: %v", err)
	}
	if repo.reclamations[9].StatutID != idReclamationRefusee {
		t.Errorf("StatutID = %d, attendu refusee (%d)", repo.reclamations[9].StatutID, idReclamationRefusee)
	}
	if repo.statutsAppliques[1] != idStatutLitige {
		t.Errorf("statut = %d, attendu litige (%d)", repo.statutsAppliques[1], idStatutLitige)
	}
}

func TestEnregistrerConstatGHResoluRendPaiementPossible(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.tickets[1] = &domain.Ticket{ID: 1}
	repo.incidents[1] = &domain.Incident{TicketID: 1, PrestataireID: int64Ptr(42)}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerConstatGH(context.Background(), 1, true); err != nil {
		t.Fatalf("EnregistrerConstatGH: %v", err)
	}
	inc := repo.incidents[1]
	if inc.ModeVerificationID == nil || *inc.ModeVerificationID != idModeVerificationGH {
		t.Errorf("ModeVerificationID = %v, attendu verification_gh (%d)", inc.ModeVerificationID, idModeVerificationGH)
	}
	if inc.VerificationResultatID == nil || *inc.VerificationResultatID != idResultatPositive {
		t.Errorf("VerificationResultatID = %v, attendu positive (%d)", inc.VerificationResultatID, idResultatPositive)
	}
	if inc.DateResolution == nil {
		t.Error("DateResolution non renseignée alors que le constat GH est positif")
	}
	if _, applique := repo.statutsAppliques[1]; applique {
		t.Errorf("statut appliqué = %d, aucun changement de statut attendu (constat positif)", repo.statutsAppliques[1])
	}
}

func TestEnregistrerConstatGHNonResoluDemarreReclamation(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.tickets[1] = &domain.Ticket{ID: 1}
	repo.incidents[1] = &domain.Incident{TicketID: 1, PrestataireID: int64Ptr(42)}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerConstatGH(context.Background(), 1, false); err != nil {
		t.Fatalf("EnregistrerConstatGH: %v", err)
	}
	inc := repo.incidents[1]
	if inc.VerificationResultatID == nil || *inc.VerificationResultatID != idResultatNegative {
		t.Errorf("VerificationResultatID = %v, attendu negative (%d)", inc.VerificationResultatID, idResultatNegative)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteTiers {
		t.Errorf("statut = %d, attendu en_attente_tiers (%d) : réclamation envoyée", repo.statutsAppliques[1], idStatutEnAttenteTiers)
	}
	var trouvee *domain.Reclamation
	for _, r := range repo.reclamations {
		if r.TicketID == 1 {
			trouvee = r
		}
	}
	if trouvee == nil {
		t.Fatal("aucune réclamation créée")
	}
}

// TestNouveauCycleVerificationEffaceAncienResultatNegatif reproduit le
// scénario réclamation acceptée -> nouvelle intervention -> nouveau rapport
// : l'ancien résultat "negative" (et sa date de résolution) ne doit pas
// survivre au nouveau cycle de vérification (cf. bug corrigé,
// SetIncidentModeVerificationNouveauCycle plutôt que SetIncidentModeVerification).
func TestNouveauCycleVerificationEffaceAncienResultatNegatif(t *testing.T) {
	repo := newFakeIncidentRepo()
	ancienneDate := time.Now().UTC()
	repo.incidents[1] = &domain.Incident{
		TicketID:               1,
		UrgenceID:              int64Ptr(idUrgenceMoyen), // pas "faible" : voie confirmation occupant
		VerificationResultatID: int64Ptr(idResultatNegative),
		DateResolution:         &ancienneDate,
	}
	repo.tickets[1] = &domain.Ticket{ID: 1}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerRapportIntervention(context.Background(), 1, "Deuxième intervention, réparation refaite."); err != nil {
		t.Fatalf("EnregistrerRapportIntervention: %v", err)
	}
	inc := repo.incidents[1]
	if inc.VerificationResultatID != nil {
		t.Errorf("VerificationResultatID = %v, attendu nil (nouveau cycle, ancien résultat négatif effacé)", inc.VerificationResultatID)
	}
	if inc.DateResolution != nil {
		t.Errorf("DateResolution = %v, attendu nil (nouveau cycle, ancienne date effacée)", inc.DateResolution)
	}
	if inc.ModeVerificationID == nil || *inc.ModeVerificationID != idModeConfirmationOccupant {
		t.Errorf("ModeVerificationID = %v, attendu confirmation_occupant (%d)", inc.ModeVerificationID, idModeConfirmationOccupant)
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

func TestCreerIncidentAuDelaDuPlafondDCreeUnDevis(t *testing.T) {
	repo := newFakeIncidentRepo()
	categorie := categorieFuiteEau()
	repo.contratActif = &repository.ContratActif{ContratID: 900, EntrepriseID: 42}
	plafondD := int64(10000)
	repo.copropriete = &domain.Copropriete{ID: 1, PlafondOrdreServiceCentimes: &plafondD} // pas de seuil B configuré
	montant := int64(50000)                                                               // au-delà du plafond D
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
	if repo.statutsAppliques[ticket.ID] != idStatutEnAttenteTiers {
		t.Errorf("statut = %d, attendu en_attente_tiers (%d) : devis demandé au prestataire", repo.statutsAppliques[ticket.ID], idStatutEnAttenteTiers)
	}
	devisListe, _ := repo.ListDevisByTicket(context.Background(), ticket.ID)
	if len(devisListe) != 1 {
		t.Fatalf("devis créés = %d, attendu 1", len(devisListe))
	}
	if devisListe[0].StatutID != idDevisEnAttente {
		t.Errorf("statut devis = %d, attendu en_attente (%d)", devisListe[0].StatutID, idDevisEnAttente)
	}
}

func TestCreerIncidentSeuilBFranchiFallbackHumain(t *testing.T) {
	repo := newFakeIncidentRepo()
	categorie := categorieFuiteEau()
	repo.contratActif = &repository.ContratActif{ContratID: 900, EntrepriseID: 42}
	plafondD := int64(10000)
	seuilB := int64(40000)
	repo.copropriete = &domain.Copropriete{ID: 1, PlafondOrdreServiceCentimes: &plafondD, SeuilBMiseEnConcurrenceCentimes: &seuilB}
	montant := int64(50000) // au-delà du seuil B : mise en concurrence requise
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
		t.Errorf("statut = %d, attendu en_attente_gestionnaire (%d) : mise en concurrence non automatisable", repo.statutsAppliques[ticket.ID], idStatutEnAttenteGestionnaire)
	}
	if devisListe, _ := repo.ListDevisByTicket(context.Background(), ticket.ID); len(devisListe) != 0 {
		t.Errorf("devis créés = %d, attendu 0 (mise en concurrence non gérée automatiquement)", len(devisListe))
	}
}

// seedTicketAvecDevisRecu prépare un ticket+incident+devis "en_attente"
// prêt à recevoir sa réponse (EnregistrerDevisRecu), pour les tests de la
// chaîne de décision post-devis.
func seedTicketAvecDevisEnAttente(repo *fakeIncidentRepo, ticketID, coproprieteID int64) int64 {
	repo.tickets[ticketID] = &domain.Ticket{ID: ticketID, CoproprieteID: coproprieteID}
	repo.incidents[ticketID] = &domain.Incident{TicketID: ticketID}
	devisID := repo.prochainDevisID
	repo.devis[devisID] = &domain.Devis{ID: devisID, TicketID: ticketID, PrestataireID: 42, StatutID: idDevisEnAttente}
	repo.prochainDevisID++
	return devisID
}

func TestEnregistrerDevisRecuAvecDelegationActive(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.copropriete = &domain.Copropriete{ID: 1}
	repo.delegation = &domain.ConseilSyndicalDelegation{ID: 1, CoproprieteID: 1, PlafondCentimes: 100000}
	devisID := seedTicketAvecDevisEnAttente(repo, 1, 1)
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerDevisRecu(context.Background(), 1, devisID, 50000); err != nil {
		t.Fatalf("EnregistrerDevisRecu: %v", err)
	}
	if repo.devis[devisID].StatutID != idDevisRecu {
		t.Errorf("statut devis = %d, attendu recu (%d)", repo.devis[devisID].StatutID, idDevisRecu)
	}
	if repo.incidents[1].DecisionCSDemandeLe == nil {
		t.Error("DecisionCSDemandeLe non renseigné : la délégation active aurait dû être détectée")
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteConseilSyndical {
		t.Errorf("statut ticket = %d, attendu en_attente_conseil_syndical (%d)", repo.statutsAppliques[1], idStatutEnAttenteConseilSyndical)
	}
}

func TestEnregistrerDevisRecuSeuilAFranchiSansDelegation(t *testing.T) {
	repo := newFakeIncidentRepo()
	seuilA := int64(20000)
	repo.copropriete = &domain.Copropriete{ID: 1, SeuilAConsultationCSCentimes: &seuilA}
	devisID := seedTicketAvecDevisEnAttente(repo, 1, 1)
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerDevisRecu(context.Background(), 1, devisID, 50000); err != nil {
		t.Fatalf("EnregistrerDevisRecu: %v", err)
	}
	if repo.incidents[1].AvisCSDemandeLe == nil {
		t.Error("AvisCSDemandeLe non renseigné : le seuil A aurait dû être détecté comme franchi")
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteConseilSyndical {
		t.Errorf("statut ticket = %d, attendu en_attente_conseil_syndical (%d)", repo.statutsAppliques[1], idStatutEnAttenteConseilSyndical)
	}
}

func TestEnregistrerDevisRecuDecisionDirecte(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.copropriete = &domain.Copropriete{ID: 1} // aucun seuil configuré : décision directe
	devisID := seedTicketAvecDevisEnAttente(repo, 1, 1)
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerDevisRecu(context.Background(), 1, devisID, 50000); err != nil {
		t.Fatalf("EnregistrerDevisRecu: %v", err)
	}
	if repo.devis[devisID].StatutID != idDevisRetenu {
		t.Errorf("statut devis = %d, attendu retenu (%d)", repo.devis[devisID].StatutID, idDevisRetenu)
	}
	if repo.incidents[1].DevisRetenuID == nil || *repo.incidents[1].DevisRetenuID != devisID {
		t.Errorf("DevisRetenuID = %v, attendu %d", repo.incidents[1].DevisRetenuID, devisID)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteTiers {
		t.Errorf("statut ticket = %d, attendu en_attente_tiers (%d)", repo.statutsAppliques[1], idStatutEnAttenteTiers)
	}
}

func TestEnregistrerDevisRecuAuDelaDuPouvoirSyndicPasseEnAG(t *testing.T) {
	repo := newFakeIncidentRepo()
	seuilSyndic := int64(20000)
	repo.copropriete = &domain.Copropriete{ID: 1, SeuilPouvoirSyndicCentimes: &seuilSyndic}
	devisID := seedTicketAvecDevisEnAttente(repo, 1, 1)
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerDevisRecu(context.Background(), 1, devisID, 50000); err != nil {
		t.Fatalf("EnregistrerDevisRecu: %v", err)
	}
	if repo.incidents[1].AGResolutionTexte == nil {
		t.Error("AGResolutionTexte non renseigné : le pouvoir ordinaire du syndic aurait dû être détecté comme dépassé")
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteAssembleeGenerale {
		t.Errorf("statut ticket = %d, attendu en_attente_assemblee_generale (%d)", repo.statutsAppliques[1], idStatutEnAttenteAssembleeGenerale)
	}
}

func TestEnregistrerDecisionCS(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.tickets[1] = &domain.Ticket{ID: 1, CoproprieteID: 1}
	repo.incidents[1] = &domain.Incident{TicketID: 1}
	repo.devis[9] = &domain.Devis{ID: 9, TicketID: 1, StatutID: idDevisRecu}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerDecisionCS(context.Background(), 1); err != nil {
		t.Fatalf("EnregistrerDecisionCS: %v", err)
	}
	if repo.incidents[1].DecisionCSRecueLe == nil {
		t.Error("DecisionCSRecueLe non renseigné")
	}
	if repo.devis[9].StatutID != idDevisRetenu {
		t.Errorf("statut devis = %d, attendu retenu (%d)", repo.devis[9].StatutID, idDevisRetenu)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteTiers {
		t.Errorf("statut ticket = %d, attendu en_attente_tiers (%d)", repo.statutsAppliques[1], idStatutEnAttenteTiers)
	}
}

func TestEnregistrerAvisCSAuDelaDuPouvoirSyndicPasseEnAG(t *testing.T) {
	repo := newFakeIncidentRepo()
	seuilSyndic := int64(20000)
	repo.copropriete = &domain.Copropriete{ID: 1, SeuilPouvoirSyndicCentimes: &seuilSyndic}
	repo.tickets[1] = &domain.Ticket{ID: 1, CoproprieteID: 1}
	repo.incidents[1] = &domain.Incident{TicketID: 1}
	montant := int64(50000)
	repo.devis[9] = &domain.Devis{ID: 9, TicketID: 1, StatutID: idDevisRecu, MontantCentimes: &montant}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerAvisCS(context.Background(), 1, "Avis favorable du conseil syndical."); err != nil {
		t.Fatalf("EnregistrerAvisCS: %v", err)
	}
	if repo.incidents[1].AvisCSTexte == nil || *repo.incidents[1].AvisCSTexte != "Avis favorable du conseil syndical." {
		t.Errorf("AvisCSTexte = %v, attendu le texte fourni", repo.incidents[1].AvisCSTexte)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteAssembleeGenerale {
		t.Errorf("statut ticket = %d, attendu en_attente_assemblee_generale (%d) : avis consultatif, ne dispense pas du seuil pouvoir syndic", repo.statutsAppliques[1], idStatutEnAttenteAssembleeGenerale)
	}
}

func TestEnregistrerResultatAGApprouveRetientLeDevis(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.tickets[1] = &domain.Ticket{ID: 1}
	repo.incidents[1] = &domain.Incident{TicketID: 1}
	repo.devis[9] = &domain.Devis{ID: 9, TicketID: 1, StatutID: idDevisRecu}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerResultatAG(context.Background(), 1, true); err != nil {
		t.Fatalf("EnregistrerResultatAG: %v", err)
	}
	if repo.incidents[1].AGResultatID == nil || *repo.incidents[1].AGResultatID != idAGApprouve {
		t.Errorf("AGResultatID = %v, attendu approuve (%d)", repo.incidents[1].AGResultatID, idAGApprouve)
	}
	if repo.devis[9].StatutID != idDevisRetenu {
		t.Errorf("statut devis = %d, attendu retenu (%d)", repo.devis[9].StatutID, idDevisRetenu)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteTiers {
		t.Errorf("statut ticket = %d, attendu en_attente_tiers (%d)", repo.statutsAppliques[1], idStatutEnAttenteTiers)
	}
}

func TestEnregistrerResultatAGRejeteAttenteGestionnaire(t *testing.T) {
	repo := newFakeIncidentRepo()
	repo.tickets[1] = &domain.Ticket{ID: 1}
	repo.incidents[1] = &domain.Incident{TicketID: 1}
	repo.devis[9] = &domain.Devis{ID: 9, TicketID: 1, StatutID: idDevisRecu}
	svc := &IncidentService{Repo: repo}

	if err := svc.EnregistrerResultatAG(context.Background(), 1, false); err != nil {
		t.Fatalf("EnregistrerResultatAG: %v", err)
	}
	if repo.incidents[1].AGResultatID == nil || *repo.incidents[1].AGResultatID != idAGRejete {
		t.Errorf("AGResultatID = %v, attendu rejete (%d)", repo.incidents[1].AGResultatID, idAGRejete)
	}
	if repo.devis[9].StatutID != idDevisRecu {
		t.Errorf("statut devis = %d, ne devrait pas changer (résolution rejetée)", repo.devis[9].StatutID)
	}
	if repo.statutsAppliques[1] != idStatutEnAttenteGestionnaire {
		t.Errorf("statut ticket = %d, attendu en_attente_gestionnaire (%d) : cas non prévu par le graphe", repo.statutsAppliques[1], idStatutEnAttenteGestionnaire)
	}
}

func int64Ptr(v int64) *int64 { return &v }
