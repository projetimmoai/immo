package service

import (
	"context"
	"fmt"
	"time"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/repository"
)

// incidentRepo est la portion de repository.Client utilisée ici — une
// interface étroite plutôt que le type concret, pour pouvoir tester avec des
// faux (cf. incident_test.go).
type incidentRepo interface {
	ActionID(ctx context.Context, description string) (int64, error)
	TicketStatutID(ctx context.Context, description string) (int64, error)
	CategorieTechniqueID(ctx context.Context, description string) (int64, error)
	NiveauUrgenceID(ctx context.Context, description string) (int64, error)
	ListCategorieTechnique(ctx context.Context) ([]domain.CategorieTechnique, error)
	ListNiveauUrgence(ctx context.Context) ([]domain.NiveauUrgence, error)
	InsertIncident(ctx context.Context, in repository.CreerIncidentInput) (*domain.Ticket, *domain.Incident, error)
	FindCoproprieteByID(ctx context.Context, id int64) (*domain.Copropriete, error)
	FindContratActif(ctx context.Context, coproprieteID, categorieTechniqueID int64) (*repository.ContratActif, error)
	SetIncidentPrestataire(ctx context.Context, ticketID, prestataireID int64) error
	UpdateTicketStatut(ctx context.Context, ticketID, statutID int64) error
	SetIncidentRapportIntervention(ctx context.Context, ticketID int64, rapport string) error
	ModeVerificationID(ctx context.Context, description string) (int64, error)
	VerificationResultatID(ctx context.Context, description string) (int64, error)
	SetIncidentModeVerification(ctx context.Context, ticketID, modeID int64, resultatID *int64) error
	SetIncidentVerificationResultat(ctx context.Context, ticketID, resultatID int64, dateResolution *time.Time) error
	FindIncidentByTicketID(ctx context.Context, ticketID int64) (*domain.Incident, error)
	FactureStatutID(ctx context.Context, description string) (int64, error)
	InsertFacture(ctx context.Context, f *domain.Facture) (*domain.Facture, error)
	ValiderFacture(ctx context.Context, factureID, statutValideeID, validePar int64, dateValidation time.Time) error
	MettreFactureEnPaiement(ctx context.Context, factureID, statutPayeeID, payePar int64, datePaiement time.Time) error
	RapprocherFacture(ctx context.Context, factureID, statutRapprocheeID, rapprochePar int64, dateRapprochement time.Time) error
	FindFactureByTicketID(ctx context.Context, ticketID int64) (*domain.Facture, error)

	FindTicketByID(ctx context.Context, id int64) (*domain.Ticket, error)
	DevisStatutID(ctx context.Context, description string) (int64, error)
	InsertDevis(ctx context.Context, d *domain.Devis) (*domain.Devis, error)
	EnregistrerReceptionDevis(ctx context.Context, devisID, statutRecuID, montantCentimes int64, dateReception time.Time) error
	MarquerDevisStatut(ctx context.Context, devisID, statutID int64) error
	ListDevisByTicket(ctx context.Context, ticketID int64) ([]*domain.Devis, error)
	FindDelegationActive(ctx context.Context, coproprieteID, montantCentimes int64) (*domain.ConseilSyndicalDelegation, error)
	SetIncidentDevisRetenu(ctx context.Context, ticketID, devisID int64) error
	SetIncidentAvisCSDemande(ctx context.Context, ticketID int64, demandeLe time.Time) error
	SetIncidentAvisCSRecu(ctx context.Context, ticketID int64, recuLe time.Time, texte string) error
	SetIncidentDecisionCSDemande(ctx context.Context, ticketID int64, demandeLe time.Time) error
	SetIncidentDecisionCSRecue(ctx context.Context, ticketID int64, recueLe time.Time) error
	SetIncidentAGResolution(ctx context.Context, ticketID int64, texte string, inscriteLe time.Time) error
	SetIncidentAGResultat(ctx context.Context, ticketID, resultatID int64, voteeLe time.Time) error
	AGResultatID(ctx context.Context, description string) (int64, error)

	SetIncidentModeVerificationNouveauCycle(ctx context.Context, ticketID, modeID int64) error
	ReclamationStatutID(ctx context.Context, description string) (int64, error)
	InsertReclamation(ctx context.Context, r *domain.Reclamation) (*domain.Reclamation, error)
	EnregistrerReponseReclamation(ctx context.Context, reclamationID, statutID int64, dateReponse time.Time, reponseTexte string) error
}

// incidentQualifieur est la portion de claudeapi.Client utilisée ici — une
// interface étroite pour pouvoir tester avec un faux.
type incidentQualifieur interface {
	QualifierIncident(ctx context.Context, categories []domain.CategorieTechnique, urgences []domain.NiveauUrgence, objet, corpsTexte string) (claudeapi.QualificationIncident, error)
}

// IncidentService orchestre le cycle de vie d'un Incident — la tranche
// verticale du graphe de cycle de vie d'un incident (cf.
// docs/cycle-vie-incident.md) : création qualifiée, sélection du
// prestataire quand un contrat actif existe, application de la chaîne des
// seuils légaux (plafond D, devis, enveloppe C, seuil A, pouvoir ordinaire
// du syndic, vote en AG), suivi de l'intervention, vérification (voie
// confirmation occupant ou jugée inutile), facture et mise en paiement
// gatée sur la vérification, rapprochement comptable et clôture.
//
// Mise en concurrence (seuil B) simplifiée : un seul prestataire est jamais
// disponible automatiquement (contrat actif, pas de répertoire par zone
// d'intervention), donc dès que ≥2 devis distincts sont requis, le ticket
// reste en attente d'un gestionnaire humain plutôt que d'inventer un second
// prestataire.
//
// Volontairement hors de cette tranche (laissés pour un prochain jalon) :
// résolution effective d'un litige (phase 5.3.5 s'arrête à l'état
// "litige", traité par un futur graphe dédié), sélection d'un
// prestataire dans un répertoire par zone d'intervention (pas encore
// modélisé), rédaction réelle des communications (demande de devis,
// d'avis, résolution AG, réclamation — de simples enregistrements pour
// l'instant, aucun envoi réel), et l'automatisation des relances (aucun
// worker périodique pour l'instant : les étapes "en attente" sont
// enregistrées mais ne sont pas relancées automatiquement).
type IncidentService struct {
	Repo   incidentRepo
	Claude incidentQualifieur
}

// CreerIncidentInput regroupe ce que l'appelant (cf. internal/email,
// traiterIncident) sait déjà au moment de la création : la source du
// ticket, la copropriété retenue, et le contenu du signalement.
type CreerIncidentInput struct {
	SourceID      int64  // FK -> ticket_source.id, NOT NULL
	CoproprieteID int64  // FK -> copropriete.id, NOT NULL
	LotID         *int64 // FK -> lot.id ; nil = partie commune (pas encore résolu par cette tranche, cf. doc)
	ParentID      *int64 // FK -> ticket.id, si ce signalement fait suite à un autre ticket
	CreePar       *int64 // FK -> personne.id (gestionnaire à l'origine, nil si créé automatiquement depuis un signalement occupant)
	Objet         string
	CorpsTexte    string
}

// CreerIncident couvre les phases 1.10 (création du ticket), 2 (qualification)
// et le début de la phase 3 (sélection du prestataire, comparaison au
// plafond D) du graphe.
//
// Sélection du prestataire simplifiée pour cette tranche (cf.
// repository.FindContratActif) : seul un contrat de maintenance actif est
// cherché — pas de répertoire de prestataires par zone d'intervention. Si
// aucun contrat actif n'existe, le ticket reste en statut
// "en_attente_gestionnaire" : un humain doit reprendre la main. Si le
// montant estimé dépasse le plafond D, la chaîne des seuils légaux prend le
// relais (cf. demarrerDevis).
func (s *IncidentService) CreerIncident(ctx context.Context, in CreerIncidentInput) (*domain.Ticket, *domain.Incident, error) {
	if in.SourceID <= 0 {
		return nil, nil, fmt.Errorf("service: création incident: source_id est obligatoire")
	}
	if in.CoproprieteID <= 0 {
		return nil, nil, fmt.Errorf("service: création incident: copropriete_id est obligatoire")
	}

	actionID, err := s.Repo.ActionID(ctx, domain.ActionIncident)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: résolution de l'action: %w", err)
	}
	statutNouveauID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutNouveau)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: résolution du statut initial: %w", err)
	}

	categories, err := s.Repo.ListCategorieTechnique(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: chargement des catégories techniques: %w", err)
	}
	urgences, err := s.Repo.ListNiveauUrgence(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: chargement des niveaux d'urgence: %w", err)
	}

	qualif, err := s.Claude.QualifierIncident(ctx, categories, urgences, in.Objet, in.CorpsTexte)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: qualification: %w", err)
	}

	var categorieTechniqueID *int64
	if qualif.CategorieTechnique != nil {
		id, err := s.Repo.CategorieTechniqueID(ctx, *qualif.CategorieTechnique)
		if err != nil {
			return nil, nil, fmt.Errorf("service: création incident: résolution de la catégorie technique %q: %w", *qualif.CategorieTechnique, err)
		}
		categorieTechniqueID = &id
	}
	urgenceID, err := s.Repo.NiveauUrgenceID(ctx, qualif.Urgence)
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: résolution du niveau d'urgence %q: %w", qualif.Urgence, err)
	}

	ticket, incident, err := s.Repo.InsertIncident(ctx, repository.CreerIncidentInput{
		ActionID:              actionID,
		StatutID:              statutNouveauID,
		SourceID:              in.SourceID,
		CoproprieteID:         in.CoproprieteID,
		LotID:                 in.LotID,
		ParentID:              in.ParentID,
		CreePar:               in.CreePar,
		CategorieTechniqueID:  categorieTechniqueID,
		UrgenceID:             &urgenceID,
		MontantEstimeCentimes: qualif.MontantEstimeCentimes,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("service: création incident: %w", err)
	}

	if err := s.selectionnerPrestataireEtDemarrer(ctx, ticket, incident, categorieTechniqueID); err != nil {
		return ticket, incident, fmt.Errorf("service: création incident %s: sélection du prestataire: %w", ticket.Reference, err)
	}
	return ticket, incident, nil
}

// selectionnerPrestataireEtDemarrer couvre la phase 3.3 (sélection du
// prestataire, simplifiée : contrat actif seulement) puis la comparaison au
// plafond D (phase 3.4.1). Ne retourne une erreur que pour un problème
// technique (accès base) — l'absence de prestataire ou un montant au-delà
// du plafond D ne sont pas des erreurs : ils laissent le ticket en attente
// d'un gestionnaire humain (cf. doc du champ StatutID correspondant).
func (s *IncidentService) selectionnerPrestataireEtDemarrer(ctx context.Context, ticket *domain.Ticket, incident *domain.Incident, categorieTechniqueID *int64) error {
	statutAttenteGestionnaireID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteGestionnaire)
	if err != nil {
		return fmt.Errorf("résolution du statut en_attente_gestionnaire: %w", err)
	}

	if categorieTechniqueID == nil {
		// Catégorie indéterminée : impossible de chercher un contrat par
		// catégorie technique — recherche humaine du prestataire.
		return s.Repo.UpdateTicketStatut(ctx, ticket.ID, statutAttenteGestionnaireID)
	}

	contrat, err := s.Repo.FindContratActif(ctx, ticket.CoproprieteID, *categorieTechniqueID)
	if err != nil {
		return fmt.Errorf("recherche d'un contrat actif: %w", err)
	}
	if contrat == nil {
		// Phase 3.3.3 : aucun prestataire du répertoire ne correspond (ici,
		// aucun contrat actif) → recherche humaine.
		return s.Repo.UpdateTicketStatut(ctx, ticket.ID, statutAttenteGestionnaireID)
	}
	if err := s.Repo.SetIncidentPrestataire(ctx, ticket.ID, contrat.EntrepriseID); err != nil {
		return fmt.Errorf("enregistrement du prestataire retenu: %w", err)
	}
	incident.PrestataireID = &contrat.EntrepriseID

	// Phase 3.4.1 : comparaison au plafond D (ordre de service).
	cop, err := s.Repo.FindCoproprieteByID(ctx, ticket.CoproprieteID)
	if err != nil {
		return fmt.Errorf("chargement de la copropriete: %w", err)
	}
	depasse := cop == nil || cop.PlafondOrdreServiceCentimes == nil ||
		incident.MontantEstimeCentimes == nil ||
		*incident.MontantEstimeCentimes > *cop.PlafondOrdreServiceCentimes
	if depasse {
		// Au-delà du plafond D (ou plafond/montant inconnu) : un devis est
		// nécessaire (phase 3.4.2 et suivantes).
		return s.demarrerDevis(ctx, ticket, cop, incident)
	}

	// Sous le plafond D : intervention immédiate, sans devis préalable
	// (phase 3.4.1) → suivi de l'intervention, en attente du prestataire.
	return s.passerEnSuiviIntervention(ctx, ticket.ID)
}

// passerEnSuiviIntervention fait passer le ticket en phase 4 (suivi de
// l'intervention, en attente du prestataire) — point de convergence commun
// à toute décision de traitement, quelle que soit l'instance qui a tranché
// (sous le plafond D directement, syndic/IA après devis, conseil syndical
// par délégation, ou vote en AG).
func (s *IncidentService) passerEnSuiviIntervention(ctx context.Context, ticketID int64) error {
	statutAttenteTiersID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteTiers)
	if err != nil {
		return fmt.Errorf("résolution du statut en_attente_tiers: %w", err)
	}
	return s.Repo.UpdateTicketStatut(ctx, ticketID, statutAttenteTiersID)
}

// demarrerDevis couvre la phase 3.4.2-3.4.6 : comparaison au seuil B (mise
// en concurrence) pour savoir combien de devis sont requis, puis demande de
// devis. La mise en concurrence réelle (≥2 devis de prestataires distincts)
// n'est pas automatisable dans cette tranche (cf. doc de IncidentService) :
// un montant inconnu, un seuil B franchi, ou l'absence de prestataire
// laissent le ticket en attente d'un gestionnaire humain.
func (s *IncidentService) demarrerDevis(ctx context.Context, ticket *domain.Ticket, cop *domain.Copropriete, incident *domain.Incident) error {
	statutAttenteGestionnaireID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteGestionnaire)
	if err != nil {
		return fmt.Errorf("résolution du statut en_attente_gestionnaire: %w", err)
	}
	if incident.MontantEstimeCentimes == nil || incident.PrestataireID == nil {
		return s.Repo.UpdateTicketStatut(ctx, ticket.ID, statutAttenteGestionnaireID)
	}
	miseEnConcurrenceRequise := cop != nil && cop.SeuilBMiseEnConcurrenceCentimes != nil &&
		*incident.MontantEstimeCentimes > *cop.SeuilBMiseEnConcurrenceCentimes
	if miseEnConcurrenceRequise {
		// Seuil B franchi : ≥2 devis de prestataires distincts requis — la
		// sélection automatique (contrat actif) n'en fournit qu'un seul.
		return s.Repo.UpdateTicketStatut(ctx, ticket.ID, statutAttenteGestionnaireID)
	}

	statutEnAttenteID, err := s.Repo.DevisStatutID(ctx, domain.DevisStatutEnAttente)
	if err != nil {
		return fmt.Errorf("résolution du statut devis en_attente: %w", err)
	}
	maintenant := time.Now().UTC()
	if _, err := s.Repo.InsertDevis(ctx, &domain.Devis{
		TicketID:      ticket.ID,
		PrestataireID: *incident.PrestataireID,
		StatutID:      statutEnAttenteID,
		DateDemande:   &maintenant,
	}); err != nil {
		return fmt.Errorf("demande de devis: %w", err)
	}
	return s.passerEnSuiviIntervention(ctx, ticket.ID)
}

// EnregistrerDevisRecu couvre la phase 3.4.6 : réception et extraction du
// montant d'un devis, puis déclenche la suite de la chaîne de décision (cf.
// appliquerDecisionApresDevis).
func (s *IncidentService) EnregistrerDevisRecu(ctx context.Context, ticketID, devisID, montantCentimes int64) error {
	statutRecuID, err := s.Repo.DevisStatutID(ctx, domain.DevisStatutRecu)
	if err != nil {
		return fmt.Errorf("service: réception devis id=%d: résolution du statut recu: %w", devisID, err)
	}
	if err := s.Repo.EnregistrerReceptionDevis(ctx, devisID, statutRecuID, montantCentimes, time.Now().UTC()); err != nil {
		return fmt.Errorf("service: réception devis id=%d: %w", devisID, err)
	}
	if err := s.appliquerDecisionApresDevis(ctx, ticketID, montantCentimes); err != nil {
		return fmt.Errorf("service: réception devis id=%d: décision: %w", devisID, err)
	}
	return nil
}

// appliquerDecisionApresDevis couvre la phase 3.4.7-3.4.17 : une fois le
// montant confirmé par le devis, applique dans l'ordre l'enveloppe C
// (délégation active au conseil syndical), puis le seuil A (consultation du
// conseil syndical), puis compare au pouvoir ordinaire du syndic (cf.
// evaluerPouvoirSyndicEtDecider). Chaque instance saisie (CS ou AG) suspend
// la chaîne jusqu'à ce qu'un appel ultérieur (EnregistrerDecisionCS,
// EnregistrerAvisCS, EnregistrerResultatAG) la reprenne.
func (s *IncidentService) appliquerDecisionApresDevis(ctx context.Context, ticketID, montantCentimes int64) error {
	ticket, err := s.Repo.FindTicketByID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("chargement du ticket: %w", err)
	}
	if ticket == nil {
		return fmt.Errorf("ticket_id=%d introuvable", ticketID)
	}
	cop, err := s.Repo.FindCoproprieteByID(ctx, ticket.CoproprieteID)
	if err != nil {
		return fmt.Errorf("chargement de la copropriete: %w", err)
	}

	delegation, err := s.Repo.FindDelegationActive(ctx, ticket.CoproprieteID, montantCentimes)
	if err != nil {
		return fmt.Errorf("recherche d'une délégation CS active: %w", err)
	}
	if delegation != nil {
		// Enveloppe C : le conseil syndical décide et choisit lui-même
		// l'artisan — le syndic/IA n'est qu'exécutant et payeur.
		maintenant := time.Now().UTC()
		if err := s.Repo.SetIncidentDecisionCSDemande(ctx, ticketID, maintenant); err != nil {
			return fmt.Errorf("transmission du dossier au CS: %w", err)
		}
		statutID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteConseilSyndical)
		if err != nil {
			return fmt.Errorf("résolution du statut en_attente_conseil_syndical: %w", err)
		}
		return s.Repo.UpdateTicketStatut(ctx, ticketID, statutID)
	}

	seuilAFranchi := cop != nil && cop.SeuilAConsultationCSCentimes != nil && montantCentimes > *cop.SeuilAConsultationCSCentimes
	if seuilAFranchi {
		// Seuil A : simple avis consultatif du CS, le syndic/IA reste
		// décisionnaire — cf. EnregistrerAvisCS pour la suite.
		maintenant := time.Now().UTC()
		if err := s.Repo.SetIncidentAvisCSDemande(ctx, ticketID, maintenant); err != nil {
			return fmt.Errorf("demande d'avis au CS: %w", err)
		}
		statutID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteConseilSyndical)
		if err != nil {
			return fmt.Errorf("résolution du statut en_attente_conseil_syndical: %w", err)
		}
		return s.Repo.UpdateTicketStatut(ctx, ticketID, statutID)
	}

	// Ni délégation active ni seuil A franchi : comparaison directe au
	// pouvoir ordinaire du syndic (phase 3.4.17).
	return s.evaluerPouvoirSyndicEtDecider(ctx, ticket, cop, montantCentimes)
}

// evaluerPouvoirSyndicEtDecider couvre la phase 3.4.17-3.4.18 : au-delà du
// pouvoir ordinaire du syndic, une résolution est inscrite à l'ordre du
// jour de l'assemblée générale (cf. EnregistrerResultatAG pour la suite) ;
// sinon, le syndic/IA décide directement et retient le devis reçu.
func (s *IncidentService) evaluerPouvoirSyndicEtDecider(ctx context.Context, ticket *domain.Ticket, cop *domain.Copropriete, montantCentimes int64) error {
	depasseSyndic := cop != nil && cop.SeuilPouvoirSyndicCentimes != nil && montantCentimes > *cop.SeuilPouvoirSyndicCentimes
	if depasseSyndic {
		maintenant := time.Now().UTC()
		texte := fmt.Sprintf("Résolution : autorisation de la dépense pour le ticket %s (montant %d centimes).", ticket.Reference, montantCentimes)
		if err := s.Repo.SetIncidentAGResolution(ctx, ticket.ID, texte, maintenant); err != nil {
			return fmt.Errorf("rédaction de la résolution AG: %w", err)
		}
		statutID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteAssembleeGenerale)
		if err != nil {
			return fmt.Errorf("résolution du statut en_attente_assemblee_generale: %w", err)
		}
		return s.Repo.UpdateTicketStatut(ctx, ticket.ID, statutID)
	}

	// Dans les limites du pouvoir ordinaire du syndic : décision directe.
	devis, err := s.findDevisRecu(ctx, ticket.ID)
	if err != nil {
		return fmt.Errorf("recherche du devis reçu: %w", err)
	}
	return s.retenirDevis(ctx, ticket.ID, devis.ID)
}

// EnregistrerDecisionCS couvre la phase 3.4.11 : le conseil syndical, dans
// le cadre de sa délégation (enveloppe C), a voté et choisi l'artisan — le
// syndic/IA n'est plus qu'exécutant et payeur de cette décision. Le devis
// concerné est retrouvé automatiquement (cf. findDevisRecu) : cette tranche
// ne gère jamais plus d'un devis reçu à la fois par ticket.
func (s *IncidentService) EnregistrerDecisionCS(ctx context.Context, ticketID int64) error {
	if err := s.Repo.SetIncidentDecisionCSRecue(ctx, ticketID, time.Now().UTC()); err != nil {
		return fmt.Errorf("service: décision CS ticket_id=%d: %w", ticketID, err)
	}
	devis, err := s.findDevisRecu(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: décision CS ticket_id=%d: recherche du devis reçu: %w", ticketID, err)
	}
	return s.retenirDevis(ctx, ticketID, devis.ID)
}

// EnregistrerAvisCS couvre la phase 3.4.16 : réception de l'avis écrit du
// conseil syndical (seuil A) — consultatif, il ne dispense pas de comparer
// ensuite au pouvoir ordinaire du syndic (phase 3.4.17).
func (s *IncidentService) EnregistrerAvisCS(ctx context.Context, ticketID int64, texte string) error {
	if err := s.Repo.SetIncidentAvisCSRecu(ctx, ticketID, time.Now().UTC(), texte); err != nil {
		return fmt.Errorf("service: avis CS ticket_id=%d: %w", ticketID, err)
	}
	ticket, err := s.Repo.FindTicketByID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: avis CS ticket_id=%d: chargement du ticket: %w", ticketID, err)
	}
	if ticket == nil {
		return fmt.Errorf("service: avis CS ticket_id=%d: ticket introuvable", ticketID)
	}
	cop, err := s.Repo.FindCoproprieteByID(ctx, ticket.CoproprieteID)
	if err != nil {
		return fmt.Errorf("service: avis CS ticket_id=%d: chargement de la copropriete: %w", ticketID, err)
	}
	devis, err := s.findDevisRecu(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: avis CS ticket_id=%d: recherche du devis reçu: %w", ticketID, err)
	}
	montant := devis.MontantCentimes
	if montant == nil {
		return fmt.Errorf("service: avis CS ticket_id=%d: devis id=%d sans montant enregistré", ticketID, devis.ID)
	}
	if err := s.evaluerPouvoirSyndicEtDecider(ctx, ticket, cop, *montant); err != nil {
		return fmt.Errorf("service: avis CS ticket_id=%d: %w", ticketID, err)
	}
	return nil
}

// EnregistrerResultatAG couvre la phase 3.4.21-3.4.22 : enregistrement du
// résultat du vote de l'assemblée générale. Une résolution rejetée n'est
// pas couverte par le graphe (cas non prévu, hors périmètre légal simple) :
// le ticket reste en attente d'un gestionnaire humain plutôt que de
// présumer une suite.
func (s *IncidentService) EnregistrerResultatAG(ctx context.Context, ticketID int64, approuve bool) error {
	resultatDescription := domain.AGResultatRejete
	if approuve {
		resultatDescription = domain.AGResultatApprouve
	}
	resultatID, err := s.Repo.AGResultatID(ctx, resultatDescription)
	if err != nil {
		return fmt.Errorf("service: résultat AG ticket_id=%d: résolution du résultat: %w", ticketID, err)
	}
	if err := s.Repo.SetIncidentAGResultat(ctx, ticketID, resultatID, time.Now().UTC()); err != nil {
		return fmt.Errorf("service: résultat AG ticket_id=%d: %w", ticketID, err)
	}
	if !approuve {
		statutID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteGestionnaire)
		if err != nil {
			return fmt.Errorf("service: résultat AG ticket_id=%d: résolution du statut en_attente_gestionnaire: %w", ticketID, err)
		}
		return s.Repo.UpdateTicketStatut(ctx, ticketID, statutID)
	}
	devis, err := s.findDevisRecu(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: résultat AG ticket_id=%d: recherche du devis reçu: %w", ticketID, err)
	}
	return s.retenirDevis(ctx, ticketID, devis.ID)
}

// retenirDevis marque le devis choisi comme "retenu" (phase 3.4.23,
// quelle que soit l'instance qui a tranché) et fait passer le ticket en
// suivi d'intervention.
func (s *IncidentService) retenirDevis(ctx context.Context, ticketID, devisID int64) error {
	retenuID, err := s.Repo.DevisStatutID(ctx, domain.DevisStatutRetenu)
	if err != nil {
		return fmt.Errorf("résolution du statut devis retenu: %w", err)
	}
	if err := s.Repo.MarquerDevisStatut(ctx, devisID, retenuID); err != nil {
		return fmt.Errorf("marquage du devis id=%d retenu: %w", devisID, err)
	}
	if err := s.Repo.SetIncidentDevisRetenu(ctx, ticketID, devisID); err != nil {
		return fmt.Errorf("enregistrement du devis retenu sur l'incident: %w", err)
	}
	return s.passerEnSuiviIntervention(ctx, ticketID)
}

// findDevisRecu retrouve l'unique devis reçu (statut "recu") d'un ticket —
// cette tranche ne gère jamais plus d'un devis reçu à la fois par ticket
// (pas de mise en concurrence automatisée, cf. demarrerDevis).
func (s *IncidentService) findDevisRecu(ctx context.Context, ticketID int64) (*domain.Devis, error) {
	statutRecuID, err := s.Repo.DevisStatutID(ctx, domain.DevisStatutRecu)
	if err != nil {
		return nil, fmt.Errorf("résolution du statut devis recu: %w", err)
	}
	liste, err := s.Repo.ListDevisByTicket(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("listage des devis: %w", err)
	}
	for _, d := range liste {
		if d.StatutID == statutRecuID {
			return d, nil
		}
	}
	return nil, fmt.Errorf("aucun devis au statut recu pour le ticket_id=%d", ticketID)
}

// EnregistrerRapportIntervention couvre la phase 4.5 (rapport de fin
// d'intervention du prestataire) puis la phase 5.0 (choix du mode de
// vérification).
//
// Choix du mode de vérification simplifié pour cette tranche : une règle
// déterministe (urgence faible → jugée inutile, sinon confirmation
// demandée à l'occupant) plutôt qu'un appel à Claude — à raffiner dans un
// prochain jalon (cf. doc, phase 5.0.1, qui prévoit aussi une vérification
// humaine sur place pour les cas incertains, pas encore implémentée).
func (s *IncidentService) EnregistrerRapportIntervention(ctx context.Context, ticketID int64, rapport string) error {
	if err := s.Repo.SetIncidentRapportIntervention(ctx, ticketID, rapport); err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: %w", ticketID, err)
	}

	incident, err := s.Repo.FindIncidentByTicketID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: chargement de l'incident: %w", ticketID, err)
	}
	if incident == nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: incident introuvable", ticketID)
	}

	jugeeBenigne := incident.UrgenceID != nil
	if jugeeBenigne {
		faibleID, err := s.Repo.NiveauUrgenceID(ctx, domain.NiveauUrgenceFaible)
		if err != nil {
			return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du niveau faible: %w", ticketID, err)
		}
		jugeeBenigne = *incident.UrgenceID == faibleID
	}

	if jugeeBenigne {
		modeID, err := s.Repo.ModeVerificationID(ctx, domain.ModeVerificationJugeeInutile)
		if err != nil {
			return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du mode jugée inutile: %w", ticketID, err)
		}
		positiveID, err := s.Repo.VerificationResultatID(ctx, domain.VerificationResultatPositive)
		if err != nil {
			return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du résultat positif: %w", ticketID, err)
		}
		if err := s.Repo.SetIncidentModeVerification(ctx, ticketID, modeID, &positiveID); err != nil {
			return fmt.Errorf("service: rapport d'intervention ticket_id=%d: enregistrement du mode de vérification: %w", ticketID, err)
		}
		maintenant := time.Now().UTC()
		return s.Repo.SetIncidentVerificationResultat(ctx, ticketID, positiveID, &maintenant)
	}

	modeID, err := s.Repo.ModeVerificationID(ctx, domain.ModeVerificationConfirmationOccupant)
	if err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du mode confirmation occupant: %w", ticketID, err)
	}
	// SetIncidentModeVerificationNouveauCycle (et non SetIncidentModeVerification)
	// : efface explicitement un résultat/date de résolution d'un cycle de
	// vérification précédent, au cas où ce rapport fait suite à une
	// réclamation acceptée (phase 5.3.4) plutôt qu'au tout premier cycle.
	if err := s.Repo.SetIncidentModeVerificationNouveauCycle(ctx, ticketID, modeID); err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: enregistrement du mode de vérification: %w", ticketID, err)
	}
	statutAttenteEmetteurID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteEmetteur)
	if err != nil {
		return fmt.Errorf("service: rapport d'intervention ticket_id=%d: résolution du statut en_attente_emetteur: %w", ticketID, err)
	}
	return s.Repo.UpdateTicketStatut(ctx, ticketID, statutAttenteEmetteurID)
}

// EnregistrerConfirmationOccupant couvre la phase 5.1.3/5.1.5 : la voie
// confirmation de l'occupant. Une non-résolution laisse le ticket en
// attente d'un gestionnaire humain — la réclamation auprès du prestataire
// (phase 5.3) n'est pas encore implémentée dans cette tranche.
func (s *IncidentService) EnregistrerConfirmationOccupant(ctx context.Context, ticketID int64, resolu bool) error {
	resultatDescription := domain.VerificationResultatNegative
	if resolu {
		resultatDescription = domain.VerificationResultatPositive
	}
	resultatID, err := s.Repo.VerificationResultatID(ctx, resultatDescription)
	if err != nil {
		return fmt.Errorf("service: confirmation occupant ticket_id=%d: résolution du résultat: %w", ticketID, err)
	}

	var dateResolution *time.Time
	if resolu {
		maintenant := time.Now().UTC()
		dateResolution = &maintenant
	}
	if err := s.Repo.SetIncidentVerificationResultat(ctx, ticketID, resultatID, dateResolution); err != nil {
		return fmt.Errorf("service: confirmation occupant ticket_id=%d: %w", ticketID, err)
	}

	if resolu {
		// Vérification positive : la mise en paiement de la facture devient
		// possible (cf. MettreEnPaiement), mais rien à faire ici sur le
		// statut — la facture suit son propre circuit (cf. EnregistrerFacture).
		return nil
	}

	// Non-résolution (phase 5.1.5) : réclamation auprès du prestataire
	// (phase 5.3).
	if err := s.demarrerReclamation(ctx, ticketID); err != nil {
		return fmt.Errorf("service: confirmation occupant ticket_id=%d: %w", ticketID, err)
	}
	return nil
}

// EnregistrerConstatGH couvre la phase 5.2.1-5.2.3 : voie vérification par
// un gestionnaire humain sur place — déplacement et constat physique, non
// délégable (à la différence de la voie occupant, ce n'est jamais une
// simple confirmation demandée en amont : le mode de vérification est
// enregistré ici, au moment du constat lui-même). resolu=false déclenche la
// réclamation (phase 5.3), comme pour la voie occupant (cf.
// EnregistrerConfirmationOccupant) ; resolu=true rend la mise en paiement
// possible (cf. MettreEnPaiement), sans rien changer au statut ici — la
// facture suit son propre circuit.
func (s *IncidentService) EnregistrerConstatGH(ctx context.Context, ticketID int64, resolu bool) error {
	modeID, err := s.Repo.ModeVerificationID(ctx, domain.ModeVerificationGH)
	if err != nil {
		return fmt.Errorf("service: constat GH ticket_id=%d: résolution du mode: %w", ticketID, err)
	}
	resultatDescription := domain.VerificationResultatNegative
	if resolu {
		resultatDescription = domain.VerificationResultatPositive
	}
	resultatID, err := s.Repo.VerificationResultatID(ctx, resultatDescription)
	if err != nil {
		return fmt.Errorf("service: constat GH ticket_id=%d: résolution du résultat: %w", ticketID, err)
	}
	if err := s.Repo.SetIncidentModeVerification(ctx, ticketID, modeID, &resultatID); err != nil {
		return fmt.Errorf("service: constat GH ticket_id=%d: enregistrement du mode/résultat: %w", ticketID, err)
	}

	if resolu {
		maintenant := time.Now().UTC()
		if err := s.Repo.SetIncidentVerificationResultat(ctx, ticketID, resultatID, &maintenant); err != nil {
			return fmt.Errorf("service: constat GH ticket_id=%d: enregistrement de la date de résolution: %w", ticketID, err)
		}
		// Vérification positive : la mise en paiement de la facture devient
		// possible (cf. MettreEnPaiement), mais rien à faire ici sur le
		// statut — la facture suit son propre circuit (cf. EnregistrerFacture).
		return nil
	}

	// Non-résolution (phase 5.2.3) : réclamation auprès du prestataire
	// (phase 5.3), comme pour la voie occupant.
	if err := s.demarrerReclamation(ctx, ticketID); err != nil {
		return fmt.Errorf("service: constat GH ticket_id=%d: %w", ticketID, err)
	}
	return nil
}

// demarrerReclamation couvre la phase 5.3.1 : réclamation rédigée et
// adressée au prestataire, qu'elle vienne d'une non-résolution signalée par
// l'occupant (phase 5.1.5, cf. EnregistrerConfirmationOccupant) ou d'un
// constat du gestionnaire humain sur place (phase 5.2.3, cf.
// EnregistrerConstatGH). Sans prestataire connu sur l'incident (ne devrait
// pas arriver à ce stade du graphe), le ticket reste en attente d'un
// gestionnaire humain plutôt que d'échouer.
func (s *IncidentService) demarrerReclamation(ctx context.Context, ticketID int64) error {
	incident, err := s.Repo.FindIncidentByTicketID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("chargement de l'incident: %w", err)
	}
	if incident == nil || incident.PrestataireID == nil {
		statutID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteGestionnaire)
		if err != nil {
			return fmt.Errorf("résolution du statut en_attente_gestionnaire: %w", err)
		}
		return s.Repo.UpdateTicketStatut(ctx, ticketID, statutID)
	}

	statutEnvoyeeID, err := s.Repo.ReclamationStatutID(ctx, domain.ReclamationStatutEnvoyee)
	if err != nil {
		return fmt.Errorf("résolution du statut reclamation envoyee: %w", err)
	}
	maintenant := time.Now().UTC()
	texte := "Réclamation : le problème signalé ne semble pas résolu après votre intervention. Merci de revenir constater et corriger."
	if _, err := s.Repo.InsertReclamation(ctx, &domain.Reclamation{
		TicketID:      ticketID,
		PrestataireID: *incident.PrestataireID,
		Texte:         texte,
		StatutID:      statutEnvoyeeID,
		DateEnvoi:     &maintenant,
	}); err != nil {
		return fmt.Errorf("envoi de la réclamation: %w", err)
	}

	// En attente de la réponse du prestataire — même statut que l'attente
	// d'intervention (cf. passerEnSuiviIntervention) : dans les deux cas,
	// c'est le prestataire qui doit agir ensuite.
	statutAttenteTiersID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutEnAttenteTiers)
	if err != nil {
		return fmt.Errorf("résolution du statut en_attente_tiers: %w", err)
	}
	return s.Repo.UpdateTicketStatut(ctx, ticketID, statutAttenteTiersID)
}

// EnregistrerReponseReclamation couvre la phase 5.3.2 : réponse du
// prestataire à une réclamation. Acceptée → nouvelle intervention, retour
// phase 4 (phase 5.3.4). Refusée → litige (phase 5.3.5) : le paiement est
// déjà suspendu, la vérification de l'incident étant restée négative (cf.
// MettreEnPaiement, qui exige une vérification positive) — la résolution du
// litige lui-même est hors du périmètre de ce graphe.
func (s *IncidentService) EnregistrerReponseReclamation(ctx context.Context, ticketID, reclamationID int64, acceptee bool, reponseTexte string) error {
	statutDescription := domain.ReclamationStatutRefusee
	if acceptee {
		statutDescription = domain.ReclamationStatutAcceptee
	}
	statutID, err := s.Repo.ReclamationStatutID(ctx, statutDescription)
	if err != nil {
		return fmt.Errorf("service: réponse réclamation id=%d: résolution du statut: %w", reclamationID, err)
	}
	if err := s.Repo.EnregistrerReponseReclamation(ctx, reclamationID, statutID, time.Now().UTC(), reponseTexte); err != nil {
		return fmt.Errorf("service: réponse réclamation id=%d: %w", reclamationID, err)
	}

	if acceptee {
		return s.passerEnSuiviIntervention(ctx, ticketID)
	}

	statutLitigeID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutLitige)
	if err != nil {
		return fmt.Errorf("service: réponse réclamation id=%d: résolution du statut litige: %w", reclamationID, err)
	}
	return s.Repo.UpdateTicketStatut(ctx, ticketID, statutLitigeID)
}

// EnregistrerFacture couvre la phase 5.5.1-5.5.3 (réception et extraction de
// la facture du prestataire).
func (s *IncidentService) EnregistrerFacture(ctx context.Context, ticketID, prestataireID, montantFactureCentimes int64, dateReception time.Time, creePar *int64) (*domain.Facture, error) {
	statutRecueID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutRecue)
	if err != nil {
		return nil, fmt.Errorf("service: enregistrement facture ticket_id=%d: résolution du statut recue: %w", ticketID, err)
	}
	facture, err := s.Repo.InsertFacture(ctx, &domain.Facture{
		TicketID:               ticketID,
		PrestataireID:          prestataireID,
		MontantFactureCentimes: montantFactureCentimes,
		StatutID:               statutRecueID,
		DateReception:          &dateReception,
		CreePar:                creePar,
	})
	if err != nil {
		return nil, fmt.Errorf("service: enregistrement facture ticket_id=%d: %w", ticketID, err)
	}
	return facture, nil
}

// ValiderFacture couvre la phase 5.5.4 : validation du montant facturé face
// au devis ou au contrat.
func (s *IncidentService) ValiderFacture(ctx context.Context, factureID, validePar int64) error {
	statutValideeID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutValidee)
	if err != nil {
		return fmt.Errorf("service: validation facture id=%d: résolution du statut validee: %w", factureID, err)
	}
	if err := s.Repo.ValiderFacture(ctx, factureID, statutValideeID, validePar, time.Now().UTC()); err != nil {
		return fmt.Errorf("service: validation facture id=%d: %w", factureID, err)
	}
	return nil
}

// MettreEnPaiement couvre la phase 5.5.5 : mise en paiement, gatée sur la
// facture validée ET la vérification positive de l'incident — un travail
// mal exécuté ne doit pas être payé (exception d'inexécution, cf.
// docs/cycle-vie-incident.md §5.5).
func (s *IncidentService) MettreEnPaiement(ctx context.Context, ticketID, payePar int64) error {
	incident, err := s.Repo.FindIncidentByTicketID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: chargement de l'incident: %w", ticketID, err)
	}
	if incident == nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: incident introuvable", ticketID)
	}
	positiveID, err := s.Repo.VerificationResultatID(ctx, domain.VerificationResultatPositive)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: résolution du résultat positif: %w", ticketID, err)
	}
	if incident.VerificationResultatID == nil || *incident.VerificationResultatID != positiveID {
		return fmt.Errorf("service: mise en paiement ticket_id=%d refusée : la vérification n'est pas positive (exception d'inexécution)", ticketID)
	}

	facture, err := s.Repo.FindFactureByTicketID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: chargement de la facture: %w", ticketID, err)
	}
	if facture == nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d refusée : aucune facture enregistrée", ticketID)
	}
	statutValideeID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutValidee)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: résolution du statut validee: %w", ticketID, err)
	}
	if facture.StatutID != statutValideeID {
		return fmt.Errorf("service: mise en paiement ticket_id=%d refusée : la facture n'est pas validée", ticketID)
	}

	statutPayeeID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutPayee)
	if err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: résolution du statut payee: %w", ticketID, err)
	}
	if err := s.Repo.MettreFactureEnPaiement(ctx, facture.ID, statutPayeeID, payePar, time.Now().UTC()); err != nil {
		return fmt.Errorf("service: mise en paiement ticket_id=%d: %w", ticketID, err)
	}
	return nil
}

// RapprocherFacture couvre la phase 5.5.6 (rapprochement comptable) puis la
// phase 5.6.1 (clôture : statut "résolu"). La clôture définitive après
// période de grâce (5.6.2) et l'archivage (5.6.3) ne sont pas couverts ici :
// ils supposent une automatisation périodique, hors de cette tranche (cf.
// doc de IncidentService).
func (s *IncidentService) RapprocherFacture(ctx context.Context, ticketID, factureID, rapprochePar int64) error {
	statutRapprocheeID, err := s.Repo.FactureStatutID(ctx, domain.FactureStatutRapprochee)
	if err != nil {
		return fmt.Errorf("service: rapprochement facture id=%d: résolution du statut rapprochee: %w", factureID, err)
	}
	if err := s.Repo.RapprocherFacture(ctx, factureID, statutRapprocheeID, rapprochePar, time.Now().UTC()); err != nil {
		return fmt.Errorf("service: rapprochement facture id=%d: %w", factureID, err)
	}

	statutResoluID, err := s.Repo.TicketStatutID(ctx, domain.TicketStatutResolu)
	if err != nil {
		return fmt.Errorf("service: rapprochement facture id=%d: résolution du statut résolu: %w", factureID, err)
	}
	if err := s.Repo.UpdateTicketStatut(ctx, ticketID, statutResoluID); err != nil {
		return fmt.Errorf("service: rapprochement facture id=%d: clôture du ticket_id=%d: %w", factureID, ticketID, err)
	}
	return nil
}
