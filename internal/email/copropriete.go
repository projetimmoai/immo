package email

import (
	"context"
	"fmt"
	"log"

	"github.com/projetimmoai/immo/internal/claudeapi"
	"github.com/projetimmoai/immo/internal/domain"
)

// confianceMinimaleCopropriete est le seuil en dessous duquel une
// détermination de copropriété est considérée comme un échec
// d'identification (et consignée dans log, cf. DetermineCopropriete),
// même quand une copropriete_id a été retournée.
const confianceMinimaleCopropriete = 0.8

// ResolutionCopropriete est le résultat de la détermination de la
// copropriété concernée par un e-mail. CoproprieteID est nil quand la
// détermination n'a pas été possible (expéditeur sans copropriete connue,
// ou Claude n'a pas pu trancher) — ce n'est pas une erreur, juste un
// résultat indéterminé, à traiter selon Confiance.
type ResolutionCopropriete struct {
	CoproprieteID        *int64
	CoproprieteReference string
	Role                 *domain.Role // rôle retenu ; nil si non déterminé ou non pertinent
	Confiance            float64      // entre 0 (aucune confiance / indéterminé) et 1 (certitude)
	Raison               string
}

// coproprieteDecideur est la portion de claudeapi.Client utilisée ici — une
// interface étroite pour pouvoir tester avec un faux (cf. copropriete_test.go).
type coproprieteDecideur interface {
	DecideCopropriete(ctx context.Context, candidats []domain.CandidatCopropriete, objet, corpsTexte string) (claudeapi.DecisionCopropriete, error)
}

// coproprieteLogRepo est la portion de repository.Client utilisée ici pour
// consigner les échecs d'identification — une interface étroite pour
// pouvoir tester avec un faux (cf. copropriete_test.go).
type coproprieteLogRepo interface {
	LogTypeID(ctx context.Context, description string) (int64, error)
	InsertLog(ctx context.Context, l *domain.Log) (*domain.Log, error)
}

// DetermineCopropriete détermine à quelle copropriété un e-mail se
// rapporte, à partir du Contexte de son expéditeur déjà enrichi (cf.
// EnrichirExpediteur) et du contenu du message.
//
// Cas simple, sans appel à l'API Claude : aucune ou une seule copropriété
// candidate au total, tous rôles confondus (cf. candidatsCoproprietes).
// Cas ambigu (plusieurs coproprietes candidates) : appel à Claude pour
// déterminer (1) sous quel rôle l'e-mail semble avoir été envoyé, et (2)
// laquelle des coproprietes associées à ce rôle correspond, avec un indice
// de confiance.
//
// Expéditeur connu mais confiance finale < confianceMinimaleCopropriete
// (identification impossible ou trop incertaine) : un évènement
// domain.LogTypeCoproprieteNonIdentifiee est consigné via repo (cf.
// logEchecIdentification). Un échec de journalisation ne fait pas échouer
// la détermination elle-même — seulement loggé sur stderr (log.Printf).
func DetermineCopropriete(ctx context.Context, claude coproprieteDecideur, repo coproprieteLogRepo, ec *Contexte, objet, corpsTexte string) (ResolutionCopropriete, error) {
	if ec == nil || !ec.Connu {
		return ResolutionCopropriete{Raison: "expéditeur inconnu : rien à déterminer"}, nil
	}

	candidats := candidatsCoproprietes(ec)
	var res ResolutionCopropriete
	var err error

	switch len(candidats) {
	case 0:
		res = ResolutionCopropriete{Raison: "aucune copropriété associée à l'expéditeur"}

	case 1:
		c := candidats[0]
		res = ResolutionCopropriete{
			CoproprieteID:        &c.CoproprieteID,
			CoproprieteReference: c.CoproprieteReference,
			Confiance:            1,
			Raison:               "une seule copropriété associée à l'expéditeur",
		}
		if len(c.Roles) == 1 {
			res.Role = &c.Roles[0]
		}

	default:
		res, err = determinerViaClaude(ctx, claude, candidats, objet, corpsTexte)
	}
	if err != nil {
		return ResolutionCopropriete{}, err
	}

	if res.Confiance < confianceMinimaleCopropriete {
		logEchecIdentification(ctx, repo, ec, res)
	}
	return res, nil
}

// logEchecIdentification consigne un évènement
// domain.LogTypeCoproprieteNonIdentifiee. N'importe quel échec de
// journalisation est seulement loggé sur stderr — ne doit jamais faire
// échouer DetermineCopropriete, dont le résultat métier est déjà acquis.
func logEchecIdentification(ctx context.Context, repo coproprieteLogRepo, ec *Contexte, res ResolutionCopropriete) {
	if repo == nil || ec.Personne == nil {
		return
	}
	logTypeID, err := repo.LogTypeID(ctx, domain.LogTypeCoproprieteNonIdentifiee)
	if err != nil {
		log.Printf("email: consignation copropriete_non_identifiee (personne id=%d): recherche du log_type: %v", ec.Personne.ID, err)
		return
	}
	message := fmt.Sprintf("confiance=%.2f (seuil=%.2f) : %s", res.Confiance, confianceMinimaleCopropriete, res.Raison)
	personneID := ec.Personne.ID
	if _, err := repo.InsertLog(ctx, &domain.Log{LogTypeID: logTypeID, Message: &message, PersonneID: &personneID, CoproprieteID: res.CoproprieteID}); err != nil {
		log.Printf("email: consignation copropriete_non_identifiee (personne id=%d): %v", ec.Personne.ID, err)
	}
}

// determinerViaClaude délègue la décision à Claude (cas ambigu :
// plusieurs coproprietes candidates) et revalide sa réponse contre la
// liste des candidats fournis — Claude ne doit jamais faire foi seul sur
// un copropriete_id qu'il aurait inventé.
func determinerViaClaude(ctx context.Context, claude coproprieteDecideur, candidats []domain.CandidatCopropriete, objet, corpsTexte string) (ResolutionCopropriete, error) {
	if claude == nil {
		return ResolutionCopropriete{}, fmt.Errorf("email: détermination de copropriété ambiguë (%d candidats) mais aucun client Claude configuré", len(candidats))
	}

	decision, err := claude.DecideCopropriete(ctx, candidats, objet, corpsTexte)
	if err != nil {
		return ResolutionCopropriete{}, fmt.Errorf("email: détermination de copropriété via Claude: %w", err)
	}

	res := ResolutionCopropriete{Role: decision.Role, Confiance: decision.Confiance, Raison: decision.Raison}
	if decision.CoproprieteID == nil {
		return res, nil
	}

	reference, ok := referenceDuCandidat(candidats, *decision.CoproprieteID)
	if !ok {
		// Sécurité : Claude n'est censé choisir que parmi les candidats
		// fournis. S'il en invente un, on n'y fait pas confiance plutôt que
		// de propager un copropriete_id qui n'a pas de sens.
		return ResolutionCopropriete{
			Raison: fmt.Sprintf("réponse Claude incohérente : copropriete_id=%d ne figure pas parmi les %d candidats fournis", *decision.CoproprieteID, len(candidats)),
		}, nil
	}
	res.CoproprieteID = decision.CoproprieteID
	res.CoproprieteReference = reference
	return res, nil
}

func referenceDuCandidat(candidats []domain.CandidatCopropriete, id int64) (string, bool) {
	for _, c := range candidats {
		if c.CoproprieteID == id {
			return c.CoproprieteReference, true
		}
	}
	return "", false
}

// candidatsCoproprietes rassemble, à partir du Contexte enrichi d'un
// e-mail, toutes les coproprietes candidates pour son rattachement,
// dédupliquées par CoproprieteID (une même copropriete peut apparaître
// sous plusieurs rôles : ses Roles cumulent alors les rôles concernés).
//
// Limitation connue : Coproprietes (occupant/client) vient de Lots, qui ne
// distingue pas occupant de client au niveau du lot (cf. doc de Contexte)
// — chaque copropriete de cette liste se voit donc attribuer tous les
// rôles parmi {occupant, client} présents globalement chez la Personne,
// pas seulement celui réellement associé à ce lot précis.
func candidatsCoproprietes(ec *Contexte) []domain.CandidatCopropriete {
	index := make(map[int64]*domain.CandidatCopropriete)
	var ordre []int64

	ajouter := func(id int64, nom *string, reference string, role domain.Role) {
		c, ok := index[id]
		if !ok {
			c = &domain.CandidatCopropriete{CoproprieteID: id, CoproprieteNom: nom, CoproprieteReference: reference}
			index[id] = c
			ordre = append(ordre, id)
		}
		for _, r := range c.Roles {
			if r == role {
				return
			}
		}
		c.Roles = append(c.Roles, role)
	}

	occupant, client := ec.ARole(domain.RoleOccupant), ec.ARole(domain.RoleClient)
	if occupant || client {
		for _, cop := range ec.Coproprietes {
			if occupant {
				ajouter(cop.CoproprieteID, cop.CoproprieteNom, cop.CoproprieteReference, domain.RoleOccupant)
			}
			if client {
				ajouter(cop.CoproprieteID, cop.CoproprieteNom, cop.CoproprieteReference, domain.RoleClient)
			}
		}
	}

	for _, cop := range ec.CoproprietesGestion {
		ajouter(cop.CoproprieteID, cop.CoproprieteNom, cop.CoproprieteReference, domain.RoleGestionnaire)
	}

	for _, contrat := range ec.Contrats {
		ajouter(contrat.CoproprieteID, contrat.CoproprieteNom, contrat.CoproprieteReference, domain.RoleFournisseur)
	}

	candidats := make([]domain.CandidatCopropriete, 0, len(ordre))
	for _, id := range ordre {
		candidats = append(candidats, *index[id])
	}
	return candidats
}
