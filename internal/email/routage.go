package email

import (
	"fmt"
	"strings"

	"github.com/projetimmoai/immo/internal/domain"
)

// Decision est le résultat du routage d'un e-mail : l'action retenue (voir
// domain.Action*) et une explication lisible, pour tracer pourquoi. Point de
// départ volontairement simple (contexte de l'expéditeur + mots-clés) : à
// affiner avec de vrais cas réels — possiblement remplacé par un appel à
// l'API Claude —, et à terme brancher sur les fonctions métier
// correspondantes (création d'incident, de sinistre...). Ne détermine pas
// encore de sous_action (cf. domain.SousAction) : viendra avec l'affinage.
type Decision struct {
	Action string
	Raison string
}

// motsClesParAction associe à des actions d'e-mail des mots-clés (en
// minuscules, sans accent) recherchés dans l'objet et le corps texte. Ordre
// de priorité : la première action dont un mot-clé est trouvé l'emporte.
var motsClesParAction = []struct {
	Action   string
	MotsCles []string
}{
	{domain.ActionSinistre, []string{"sinistre", "assurance", "degat des eaux", "indemnisation", "expert"}},
	{domain.ActionAssembleeGenerale, []string{"assemblee generale", "convocation", " ag ", "ordre du jour", "proces-verbal"}},
	{domain.ActionIncident, []string{"panne", "fuite", "ascenseur", "interphone", "incident", "dysfonctionnement"}},
}

// DecideRoute décide de l'action d'un e-mail à partir de son contenu (objet
// + corps texte) et du contexte de son expéditeur (cf. EnrichirExpediteur).
// Ne déclenche aucune action : c'est au code appelant de brancher l'action
// retenue sur la fonction métier correspondante.
func DecideRoute(ctx *Contexte, objet, corpsTexte string) Decision {
	if ctx == nil || !ctx.Connu {
		return Decision{
			Action: domain.ActionIndetermine,
			Raison: "expéditeur inconnu du système : impossible de le rattacher à une copropriété ou un lot",
		}
	}

	texte := normaliser(objet + " " + corpsTexte)
	for _, entree := range motsClesParAction {
		for _, motCle := range entree.MotsCles {
			if strings.Contains(texte, motCle) {
				return Decision{
					Action: entree.Action,
					Raison: fmt.Sprintf("mot-clé %q trouvé dans l'objet ou le corps", strings.TrimSpace(motCle)),
				}
			}
		}
	}

	if ctx.ARole(domain.RoleFournisseur) && len(ctx.Contrats) > 0 {
		return Decision{
			Action: domain.ActionAutre,
			Raison: fmt.Sprintf("expéditeur fournisseur connu (%d contrat(s)), mais aucun mot-clé d'action détecté", len(ctx.Contrats)),
		}
	}

	return Decision{
		Action: domain.ActionAutre,
		Raison: "expéditeur connu, mais aucun mot-clé d'action détecté",
	}
}

// normaliser met en minuscules et retire les accents courants, pour une
// recherche de mots-clés insensible à la casse et aux accents.
func normaliser(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"à", "a", "â", "a",
		"î", "i", "ï", "i",
		"ô", "o", "ö", "o",
		"ù", "u", "û", "u", "ü", "u",
		"ç", "c",
	)
	return replacer.Replace(s)
}
