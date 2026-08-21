package claudeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/projetimmoai/immo/internal/domain"
)

// decisionActionTool est le nom de l'outil forcé utilisé pour obtenir une
// réponse structurée de Claude (cf. decisionCoproprieteTool pour le même
// principe).
const decisionActionTool = "decider_action"

// DecisionAction est la réponse structurée de Claude à DecideAction :
// l'action retenue — une des descriptions fournies dans actions, non
// revalidée ici (cf. email.routerVersActions, qui fait cette vérification
// avant de faire confiance à la réponse) —, avec un indice de confiance et
// une explication.
type DecisionAction struct {
	Action    string
	Confiance float64 // entre 0 (aucune confiance) et 1 (certitude)
	Raison    string
}

// DecideAction demande à Claude, à partir d'une liste d'actions candidates
// et du contexte de routage d'un e-mail dont la copropriété a déjà été
// identifiée (personne, rôle, copropriété, lots/contrats éventuels),
// quelle action décrit le mieux le contenu de cet e-mail.
//
// actions n'est pas forcément la table action entière : l'appelant peut
// n'en fournir qu'un sous-ensemble pertinent pour le rôle de l'expéditeur
// (cf. email.RouterOccupant, qui ne propose que les actions applicables à
// un occupant).
func (c *Client) DecideAction(ctx context.Context, actions []domain.Action, ctxRoutage domain.ContexteRoutage, objet, corpsTexte string) (DecisionAction, error) {
	descriptions := make([]string, len(actions))
	for i, a := range actions {
		descriptions[i] = a.Description
	}

	tool := anthropic.ToolParam{
		Name:        decisionActionTool,
		Description: anthropic.String("Enregistre l'action retenue pour cet e-mail, parmi la liste fournie."),
		Strict:      anthropic.Bool(true),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"action": map[string]any{
					"type":        "string",
					"enum":        descriptions,
					"description": "L'action qui décrit le mieux le contenu de cet e-mail, parmi la liste fournie.",
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
			Required:    []string{"action", "confiance", "raison"},
			ExtraFields: map[string]any{"additionalProperties": false},
		},
	}

	resp, err := c.anthropic.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     Model,
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{{
			Text: "Tu aides un cabinet de gestion de copropriétés à router ses e-mails entrants. " +
				"On te donne le contexte de l'expéditeur d'un e-mail — déjà identifié : sa " +
				"copropriété, son rôle, et ses lots ou contrats éventuels dans cette copropriété — " +
				"ainsi que le contenu de l'e-mail. Détermine, parmi la liste des actions fournies, " +
				"celle qui décrit le mieux la nature de cet e-mail. Si aucune ne correspond " +
				"clairement, choisis la plus proche plutôt que de deviner au hasard, et reflète " +
				"toujours ton incertitude dans confiance — une confiance basse est traitée séparément " +
				"(l'e-mail sera consigné pour vérification humaine), ce n'est pas une erreur de ta " +
				"part de la donner basse quand c'est justifié.",
		}},
		Tools:      []anthropic.ToolUnionParam{{OfTool: &tool}},
		ToolChoice: anthropic.ToolChoiceUnionParam{OfTool: &anthropic.ToolChoiceToolParam{Name: decisionActionTool}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(promptAction(ctxRoutage, objet, corpsTexte))),
		},
	})
	if err != nil {
		return DecisionAction{}, fmt.Errorf("claudeapi: appel de détermination de l'action: %w", err)
	}

	for _, block := range resp.Content {
		toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok || toolUse.Name != decisionActionTool {
			continue
		}
		var out struct {
			Action    string  `json:"action"`
			Confiance float64 `json:"confiance"`
			Raison    string  `json:"raison"`
		}
		if err := json.Unmarshal([]byte(toolUse.JSON.Input.Raw()), &out); err != nil {
			return DecisionAction{}, fmt.Errorf("claudeapi: décodage de la décision d'action: %w", err)
		}
		return DecisionAction{Action: out.Action, Confiance: clamp01(out.Confiance), Raison: out.Raison}, nil
	}

	return DecisionAction{}, fmt.Errorf("claudeapi: aucune décision d'action retournée (stop_reason=%s)", resp.StopReason)
}

// promptAction construit le message utilisateur envoyé à Claude : le
// contexte de routage puis le contenu de l'e-mail.
func promptAction(ctxRoutage domain.ContexteRoutage, objet, corpsTexte string) string {
	var sb strings.Builder
	sb.WriteString("Contexte de l'expéditeur :\n")
	if ctxRoutage.Personne != nil {
		fmt.Fprintf(&sb, "- Personne : %s\n", ctxRoutage.Personne.Reference)
	}
	if ctxRoutage.Role != nil {
		fmt.Fprintf(&sb, "- Rôle : %s\n", *ctxRoutage.Role)
	}
	fmt.Fprintf(&sb, "- Copropriété : %s\n", ctxRoutage.CoproprieteReference)
	if len(ctxRoutage.LotsReferences) > 0 {
		fmt.Fprintf(&sb, "- Lots : %s\n", strings.Join(ctxRoutage.LotsReferences, ", "))
	}
	if len(ctxRoutage.ContratsNumeros) > 0 {
		fmt.Fprintf(&sb, "- Contrats : %s\n", strings.Join(ctxRoutage.ContratsNumeros, ", "))
	}
	sb.WriteString("\nObjet de l'e-mail : ")
	sb.WriteString(objet)
	sb.WriteString("\n\nCorps de l'e-mail :\n")
	sb.WriteString(corpsTexte)
	return sb.String()
}
