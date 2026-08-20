package gmailapi

import (
	"encoding/base64"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

// gmailMessage reflète la forme JSON brute d'un message Gmail (format=full).
type gmailMessage struct {
	ID           string    `json:"id"`
	ThreadID     string    `json:"threadId"`
	InternalDate string    `json:"internalDate"` // epoch millisecondes, en chaîne
	Payload      gmailPart `json:"payload"`
}

type gmailPart struct {
	MimeType string        `json:"mimeType"`
	Filename string        `json:"filename"`
	Headers  []gmailHeader `json:"headers"`
	Body     gmailPartBody `json:"body"`
	Parts    []gmailPart   `json:"parts"`
}

type gmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type gmailPartBody struct {
	AttachmentID string `json:"attachmentId"`
	Size         int64  `json:"size"`
	Data         string `json:"data"` // base64url, présent seulement si le contenu est inclus (pas une pièce jointe séparée)
}

// Message est un e-mail Gmail normalisé : en-têtes extraits, corps
// texte/HTML décodés, pièces jointes listées (contenu non téléchargé ici —
// cf. Client.GetAttachment). Indépendant du format JSON brut de l'API
// Gmail, pour ne pas propager ce détail au reste du code (internal/email).
type Message struct {
	ID              string
	ThreadID        string
	MessageIDHeader string // en-tête RFC822 Message-ID, sert à dédupliquer (cf. domain.Email.MessageID)
	Date            time.Time
	From            string // adresse e-mail extraite de l'en-tête "From" (sans le nom affiché)
	Subject         string
	BodyText        string
	BodyHTML        string
	Attachments     []Attachment
}

// Attachment décrit une pièce jointe d'un Message, sans son contenu.
type Attachment struct {
	AttachmentID string // id Gmail, à passer à Client.GetAttachment pour le contenu
	Filename     string
	MimeType     string
	SizeOctets   int64
}

func parseMessage(raw gmailMessage) (*Message, error) {
	m := &Message{
		ID:       raw.ID,
		ThreadID: raw.ThreadID,
	}

	if raw.InternalDate != "" {
		ms, err := strconv.ParseInt(raw.InternalDate, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("gmailapi: internalDate invalide (%q) pour le message %s: %w", raw.InternalDate, raw.ID, err)
		}
		m.Date = time.UnixMilli(ms).UTC()
	}

	for _, h := range raw.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "message-id":
			m.MessageIDHeader = h.Value
		case "subject":
			m.Subject = h.Value
		case "from":
			if addr, err := mail.ParseAddress(h.Value); err == nil {
				m.From = addr.Address
			} else {
				m.From = h.Value // best-effort : on garde la valeur brute plutôt que d'échouer
			}
		}
	}

	if err := walkParts(raw.Payload, m); err != nil {
		return nil, fmt.Errorf("gmailapi: décodage du contenu du message %s: %w", raw.ID, err)
	}

	return m, nil
}

// walkParts parcourt récursivement l'arbre MIME d'un message et remplit
// BodyText/BodyHTML (premier morceau trouvé de chaque type) et Attachments.
func walkParts(part gmailPart, m *Message) error {
	isAttachment := part.Filename != "" || part.Body.AttachmentID != ""

	switch {
	case isAttachment:
		m.Attachments = append(m.Attachments, Attachment{
			AttachmentID: part.Body.AttachmentID,
			Filename:     part.Filename,
			MimeType:     part.MimeType,
			SizeOctets:   part.Body.Size,
		})
	case part.MimeType == "text/plain" && m.BodyText == "" && part.Body.Data != "":
		text, err := decodeBase64URL(part.Body.Data)
		if err != nil {
			return fmt.Errorf("corps text/plain: %w", err)
		}
		m.BodyText = string(text)
	case part.MimeType == "text/html" && m.BodyHTML == "" && part.Body.Data != "":
		html, err := decodeBase64URL(part.Body.Data)
		if err != nil {
			return fmt.Errorf("corps text/html: %w", err)
		}
		m.BodyHTML = string(html)
	}

	for _, child := range part.Parts {
		if err := walkParts(child, m); err != nil {
			return err
		}
	}
	return nil
}

// decodeBase64URL décode le format base64url (avec ou sans padding) utilisé
// par l'API Gmail pour le contenu des corps et pièces jointes.
func decodeBase64URL(s string) ([]byte, error) {
	if b, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(s); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(s)
}
