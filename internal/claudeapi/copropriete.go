package claudeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/projetimmoai/immo/internal/domain"
)

// decisionCoproprieteTool est le nom de l'outil forcé utilisé pour obtenir
// une réponse structurée de Claude (plutôt que du texte libre à parser) :
// cf. shared/tool-use-concepts.md de la doc Anthropic — un tool_choice
// forcé sur un unique outil garantit une sortie qui respecte le schéma.
const decisionCoproprieteTool = "decider_copropriete"

// roleIndetermine est la valeur sentinelle renvoyée par Claude pour "role"
// quand il ne peut pas trancher. Un enum JSON Schema ne peut pas mélanger
// null avec des valeurs string (l'API rejette "type": ["string","null"]
// combiné à "enum") — on utilise donc une chaîne dédiée plutôt que null.
const roleIndetermine = "indetermine"

// DecisionCopropriete est la réponse structurée de Claude à
// DecideCopropriete : le rôle et la copropriété retenus (nil si Claude n'a
// pas pu trancher), avec un indice de confiance et une explication.
// CoproprieteID n'est pas revalidé ici contre la liste des candidats — cf.
// email.DetermineCopropriete, qui fait cette vérification avant de faire
// confiance à la réponse.
type DecisionCopropriete struct {
	Role          *domain.Role
	CoproprieteID *int64
	Confiance     float64 // entre 0 (aucune confiance) et 1 (certitude)
	Raison        string
}

// DecideCopropriete demande à Claude, à partir d'une liste de coproprietes
// candidates (chacune avec le ou les rôles sous lesquels l'expéditeur d'un
// e-mail y est associé) et du contenu de l'e-mail, sous quel rôle l'e-mail
// semble avoir été envoyé et à quelle copropriété il se rapporte.
//
// N'a de sens que si candidats contient plusieurs entrées : avec 0 ou 1
// candidat la décision est triviale et ne doit pas passer par ici (cf.
// email.DetermineCopropriete, qui gère ces cas sans appel API).
func (c *Client) DecideCopropriete(ctx context.Context, candidats []domain.CandidatCopropriete, objet, corpsTexte string) (DecisionCopropriete, error) {
	tool := anthropic.ToolParam{
		Name:        decisionCoproprieteTool,
		Description: anthropic.String("Enregistre la copropriété (et le rôle de l'expéditeur) retenue pour cet e-mail, parmi les candidats fournis."),
		Strict:      anthropic.Bool(true),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"role": map[string]any{
					"type":        "string",
					"enum":        []string{"occupant", "client", "fournisseur", "gestionnaire", roleIndetermine},
					"description": "Le rôle sous lequel l'e-mail semble avoir été envoyé, ou \"" + roleIndetermine + "\" si indéterminable.",
				},
				"copropriete_id": map[string]any{
					"type":        []string{"integer", "null"},
					"description": "L'id de la copropriété concernée, parmi les candidats fournis. null si aucune ne correspond avec suffisamment de certitude.",
				},
				"confiance": map[string]any{
					"type":        "number",
					"description": "Indice de confiance dans la décision, un nombre entre 0 (aucune confiance) et 1 (certitude) inclus.",
				},
				"raison": map[string]any{
					"type":        "string",
					"description": "Explication brève (une phrase) de la décision.",
				},
			},
			Required:    []string{"role", "copropriete_id", "confiance", "raison"},
			ExtraFields: map[string]any{"additionalProperties": false},
		},
	}

	resp, err := c.anthropic.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     Model,
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{{
			Text: "Tu aides un cabinet de gestion de copropriétés à router ses e-mails entrants. " +
				"On te donne le contenu d'un e-mail et la liste des copropriétés associées à son " +
				"expéditeur, chacune avec le ou les rôles sous lesquels cet expéditeur y est connu " +
				"(occupant, client = copropriétaire, fournisseur = sous contrat, gestionnaire = membre " +
				"du cabinet en charge). Détermine sous quel rôle cet e-mail semble avoir été envoyé et " +
				"à quelle copropriété, parmi les candidats fournis, il se rapporte. Si le contenu ne " +
				"permet pas de trancher avec une confiance raisonnable, retourne null pour " +
				"copropriete_id (et \"" + roleIndetermine + "\" pour role) plutôt que de deviner, et " +
				"reflète toujours ton incertitude dans confiance.",
		}},
		Tools:      []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceUnionParam{OfTool: &anthropic.ToolChoiceToolParam{Name: decisionCoproprieteTool}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(promptCopropriete(candidats, objet, corpsTexte))),
		},
	})
	if err != nil {
		return DecisionCopropriete{}, fmt.Errorf("claudeapi: appel de détermination de copropriété: %w", err)
	}

	for _, block := range resp.Content {
		toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok || toolUse.Name != decisionCoproprieteTool {
			continue
		}
		var out struct {
			Role          string  `json:"role"`
			CoproprieteID *int64  `json:"copropriete_id"`
			Confiance     float64 `json:"confiance"`
			Raison        string  `json:"raison"`
		}
		if err := json.Unmarshal([]byte(toolUse.JSON.Input.Raw()), &out); err != nil {
			return DecisionCopropriete{}, fmt.Errorf("claudeapi: décodage de la décision de copropriété: %w", err)
		}
		d := DecisionCopropriete{CoproprieteID: out.CoproprieteID, Confiance: clamp01(out.Confiance), Raison: out.Raison}
		if out.Role != "" && out.Role != roleIndetermine {
			r := domain.Role(out.Role)
			d.Role = &r
		}
		return d, nil
	}

	return DecisionCopropriete{}, fmt.Errorf("claudeapi: aucune décision de copropriété retournée (stop_reason=%s)", resp.StopReason)
}

// clamp01 ramène une confiance dans [0, 1] : le schéma de l'outil ne peut
// pas imposer de borne côté API (minimum/maximum non supportés sur les
// schémas d'outils), donc on le fait ici plutôt que de faire confiance
// aveuglément à la valeur retournée par Claude.
func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// promptCopropriete construit le message utilisateur envoyé à Claude :
// la liste des candidats (id, référence, nom, rôles) puis le contenu de
// l'e-mail.
func promptCopropriete(candidats []domain.CandidatCopropriete, objet, corpsTexte string) string {
	var sb strings.Builder
	sb.WriteString("Coproprietes candidates :\n")
	for _, c := range candidats {
		nom := ""
		if c.CoproprieteNom != nil {
			nom = *c.CoproprieteNom
		}
		roles := make([]string, len(c.Roles))
		for i, r := range c.Roles {
			roles[i] = string(r)
		}
		fmt.Fprintf(&sb, "- id=%d, reference=%s, nom=%q, rôle(s)=%s\n", c.CoproprieteID, c.CoproprieteReference, nom, strings.Join(roles, ", "))
	}
	sb.WriteString("\nObjet de l'e-mail : ")
	sb.WriteString(objet)
	sb.WriteString("\n\nCorps de l'e-mail :\n")
	sb.WriteString(corpsTexte)
	return sb.String()
}
