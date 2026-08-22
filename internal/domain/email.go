package domain

import "time"

// Valeurs connues de action.description (voir migration 001). Ce sont des
// tables de référence éditables en base : ces constantes servent de clés de
// recherche (par description), jamais d'ID en dur.
const (
	ActionSinistre              = "sinistre"
	ActionIncident              = "incident"
	ActionAssembleeGenerale     = "assemblee_generale"
	ActionAutre                 = "autre"
	ActionComptabilite          = "comptabilite"
	ActionMutation              = "mutation"
	ActionContentieux           = "contentieux"
	ActionTravaux               = "travaux"
	ActionControleGestion       = "controle_gestion"
	ActionDemandeAdministrative = "demande_administrative"
)

// Action est la table de référence racine du routage des e-mails (sinistre,
// incident, AG, autre...), et plus largement de toute décision de routage.
type Action struct {
	ID          int64
	CreatedAt   time.Time
	Description string
}

// SousAction est une sous-catégorie d'une Action (ex: "degat_des_eaux" sous
// "sinistre"), pouvant elle-même avoir des sous-sous-actions : une
// hiérarchie récursive de profondeur arbitraire. ActionID identifie
// toujours l'Action racine de la hiérarchie (dénormalisé, valable à
// n'importe quelle profondeur) ; ParentID est nul pour une sous_action de
// premier niveau (directement sous l'Action), et pointe sinon vers sa
// sous_action parente. La Description n'est unique que parmi les
// sous_action partageant le même (ActionID, ParentID) — cf. contrainte
// UNIQUE NULLS NOT DISTINCT en base.
type SousAction struct {
	ID          int64
	CreatedAt   time.Time
	Description string
	ActionID    int64  // FK -> action.id (action racine, à toute profondeur)
	ParentID    *int64 // FK -> sous_action.id, nul si sous_action de premier niveau
}

// Email complète une TicketSource (cf. domain.TicketSource, qui porte le
// squelette commun : expéditeur résolu, copropriete/lot déterminés, statut
// de traitement...) avec les champs propres à un e-mail : son identité
// (Message-ID, adresse brute) et son contenu.
//
// Pas d'ActionID/SousActionID ici : un même e-mail peut donner lieu à
// plusieurs demandes distinctes (cf. email.routerVersActions), donc à
// plusieurs Ticket — chacun porte sa propre action (cf. domain.Ticket).
type Email struct {
	TicketSourceID  int64   // PK -> ticket_source.id (1-1, pas d'id propre)
	MessageID       *string // header RFC822 Message-ID, sert à dédupliquer
	ExpediteurEmail string
	Objet           *string
	CorpsTexte      *string
	CorpsHTML       *string
}

// EmailPieceJointe est une pièce jointe d'un Email, stockée dans Supabase
// Storage (cf. internal/storage, bucket storage.Bucket) et référencée ici
// par son chemin d'objet — pas une URL permanente : le fichier n'étant pas
// public, l'accès se fait via une URL signée temporaire générée à la
// demande (storage.Client.SignedURL), jamais stockée en base.
type EmailPieceJointe struct {
	ID             int64
	CreatedAt      time.Time
	EmailID        int64 // FK -> email.ticket_source_id
	NomFichier     string
	TypeMime       *string
	TailleOctets   *int64
	CheminStockage *string // chemin de l'objet dans storage.Bucket
}
