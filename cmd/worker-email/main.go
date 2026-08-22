// Command worker-email est, pour l'instant, un outil "à la main" (one-shot,
// pas de boucle ni de connexion Gmail) pour faire tourner de bout en bout le
// pipeline déjà construit dans internal/email : enregistrement de la
// TicketSource, enrichissement de l'expéditeur, détermination de la
// copropriété concernée, puis routage vers une action (pour l'instant, seul
// le rôle occupant a un routeur, cf. email.RouterOccupant).
//
// Le message traité est fourni via les flags (ou un exemple par défaut) —
// pas encore lu depuis Gmail : cf. internal/gmailapi, dont la connexion
// (OAuth2, cmd/gmail-auth) est un chantier volontairement séparé de celui-
// ci.
//
// Fait de vrais appels réseau et écrit réellement en base (Supabase, et
// Claude pour la détermination copropriete/action puis la qualification
// d'un incident) — nécessite SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY et
// ANTHROPIC_API_KEY dans l'environnement.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
	"github.com/projetimmoai/immo/internal/email"
	"github.com/projetimmoai/immo/internal/gmailapi"
	"github.com/projetimmoai/immo/internal/repository"
	"github.com/projetimmoai/immo/internal/service"
)

func main() {
	from := flag.String("from", "sophie.moreau@example.fr", "adresse e-mail de l'expéditeur (doit exister dans la table personne pour donner un résultat intéressant)")
	subject := flag.String("subject", "Panne d'ascenseur", "objet de l'e-mail")
	body := flag.String("body", "Bonjour, l'ascenseur de la résidence est en panne depuis ce matin, merci d'intervenir rapidement.", "corps de l'e-mail")
	flag.Parse()

	if err := run(*from, *subject, *body); err != nil {
		fmt.Fprintln(os.Stderr, "worker-email:", err)
		os.Exit(1)
	}
}

func run(from, subject, body string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 60*time.Second)
	defer cancelTimeout()

	repo, err := repository.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("initialisation du client Supabase: %w", err)
	}
	claude, err := claudeapi.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("initialisation du client Claude: %w", err)
	}

	msg := &gmailapi.Message{From: from, Subject: subject, BodyText: body}

	fmt.Printf("--- e-mail ---\nDe : %s\nObjet : %s\nCorps : %s\n\n", msg.From, msg.Subject, msg.BodyText)

	// 1. Enrichissement de l'expéditeur : identité, rôles, lots, contrats.
	ctxEmail, err := email.TraiterMessage(ctx, repo, msg)
	if err != nil {
		return fmt.Errorf("enrichissement de l'expéditeur: %w", err)
	}
	if !ctxEmail.Connu {
		fmt.Println("--- résultat ---\nExpéditeur inconnu du système : rien de plus à faire.")
		return nil
	}
	fmt.Printf("--- expéditeur ---\nPersonne id=%d (%s)\nRôles : %v\n\n", ctxEmail.Personne.ID, ctxEmail.Personne.Reference, ctxEmail.Roles)

	// 2. Détermination de la copropriété concernée.
	actions, err := repo.ListActions(ctx)
	if err != nil {
		return fmt.Errorf("chargement des actions: %w", err)
	}
	resCop, err := email.DetermineCopropriete(ctx, claude, repo, ctxEmail, subject, body)
	if err != nil {
		return fmt.Errorf("détermination de la copropriété: %w", err)
	}
	fmt.Printf("--- copropriété ---\n%+v\n\n", resCop)

	// 3. Enregistrement de la TicketSource (type "email") : tout Ticket créé
	// en aval doit y référencer sa source (cf. domain.Ticket.SourceID).
	statutNouveauID, err := repo.TicketSourceStatutTraitementID(ctx, domain.TicketSourceStatutNouveau)
	if err != nil {
		return fmt.Errorf("résolution du statut de traitement initial: %w", err)
	}
	source := &domain.TicketSource{
		DateReception:      time.Now().UTC(),
		PersonneID:         &ctxEmail.Personne.ID,
		CoproprieteID:      resCop.CoproprieteID,
		StatutTraitementID: statutNouveauID,
	}
	var messageID *string
	if msg.MessageIDHeader != "" {
		messageID = &msg.MessageIDHeader
	}
	createdSource, _, err := repo.InsertEmail(ctx, source, &domain.Email{
		MessageID:       messageID,
		ExpediteurEmail: msg.From,
		Objet:           &msg.Subject,
		CorpsTexte:      &msg.BodyText,
		CorpsHTML:       &msg.BodyHTML,
	})
	if err != nil {
		return fmt.Errorf("enregistrement de la TicketSource: %w", err)
	}
	fmt.Printf("--- ticket_source ---\nid=%d\n\n", createdSource.ID)

	ctxRoutage := email.NouveauContexteRoutage(ctxEmail, resCop, createdSource.ID)
	if ctxRoutage == nil {
		fmt.Println("--- résultat ---\nCopropriété non déterminée : pas de routage possible.")
		return nil
	}

	// 4. Routage vers une action, selon le rôle retenu — seul le rôle
	// occupant a un routeur pour l'instant (cf. email.RouterOccupant) ;
	// gestionnaire/coproprietaire/prestataire n'en ont pas encore.
	if ctxRoutage.Role == nil || *ctxRoutage.Role != domain.RoleOccupant {
		fmt.Printf("--- résultat ---\nRôle retenu %v : aucun routeur d'actions pour ce rôle pour l'instant (seul occupant en a un).\n", ctxRoutage.Role)
		return nil
	}
	deps := email.ActionDeps{Incident: &service.IncidentService{Repo: repo, Claude: claude}}
	resActions, err := email.RouterOccupant(ctx, claude, deps, actions, *ctxRoutage, subject, body)
	if err != nil {
		return fmt.Errorf("routage de l'action: %w", err)
	}
	fmt.Printf("--- action%s ---\n", pluriel(len(resActions)))
	for _, res := range resActions {
		fmt.Printf("%+v\n", res)
	}
	fmt.Println("\n(la ou les fonctions de traitement correspondantes ont été appelées ; cf. internal/email/incident.go pour ce qui est réellement implémenté, les autres restent des no-ops)")
	return nil
}

func pluriel(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}
