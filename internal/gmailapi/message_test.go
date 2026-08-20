package gmailapi

import (
	"encoding/base64"
	"testing"
	"time"
)

func b64(s string) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(s))
}

func TestParseMessageSimple(t *testing.T) {
	raw := gmailMessage{
		ID:           "msg1",
		ThreadID:     "thread1",
		InternalDate: "1700000000000",
		Payload: gmailPart{
			MimeType: "text/plain",
			Headers: []gmailHeader{
				{Name: "Message-Id", Value: "<abc123@example.com>"},
				{Name: "Subject", Value: "Fuite d'eau dans le lot 12"},
				{Name: "From", Value: "Jean Dupont <jean.dupont@example.com>"},
			},
			Body: gmailPartBody{Data: b64("Bonjour, il y a une fuite.")},
		},
	}

	m, err := parseMessage(raw)
	if err != nil {
		t.Fatalf("parseMessage: %v", err)
	}
	if m.ID != "msg1" || m.ThreadID != "thread1" {
		t.Errorf("ID/ThreadID = %q/%q, attendu msg1/thread1", m.ID, m.ThreadID)
	}
	if m.MessageIDHeader != "<abc123@example.com>" {
		t.Errorf("MessageIDHeader = %q", m.MessageIDHeader)
	}
	if m.Subject != "Fuite d'eau dans le lot 12" {
		t.Errorf("Subject = %q", m.Subject)
	}
	if m.From != "jean.dupont@example.com" {
		t.Errorf("From = %q, attendu l'adresse seule extraite du header", m.From)
	}
	if m.BodyText != "Bonjour, il y a une fuite." {
		t.Errorf("BodyText = %q", m.BodyText)
	}
	wantDate := time.UnixMilli(1700000000000).UTC()
	if !m.Date.Equal(wantDate) {
		t.Errorf("Date = %v, attendu %v", m.Date, wantDate)
	}
}

func TestParseMessageMultipartAvecPieceJointe(t *testing.T) {
	raw := gmailMessage{
		ID: "msg2",
		Payload: gmailPart{
			MimeType: "multipart/mixed",
			Headers: []gmailHeader{
				{Name: "From", Value: "contact@fournisseur.example"},
			},
			Parts: []gmailPart{
				{
					MimeType: "multipart/alternative",
					Parts: []gmailPart{
						{MimeType: "text/plain", Body: gmailPartBody{Data: b64("version texte")}},
						{MimeType: "text/html", Body: gmailPartBody{Data: b64("<p>version html</p>")}},
					},
				},
				{
					MimeType: "application/pdf",
					Filename: "facture.pdf",
					Body:     gmailPartBody{AttachmentID: "att1", Size: 1234},
				},
			},
		},
	}

	m, err := parseMessage(raw)
	if err != nil {
		t.Fatalf("parseMessage: %v", err)
	}
	if m.From != "contact@fournisseur.example" {
		t.Errorf("From = %q", m.From)
	}
	if m.BodyText != "version texte" {
		t.Errorf("BodyText = %q", m.BodyText)
	}
	if m.BodyHTML != "<p>version html</p>" {
		t.Errorf("BodyHTML = %q", m.BodyHTML)
	}
	if len(m.Attachments) != 1 {
		t.Fatalf("Attachments = %+v, attendu 1 pièce jointe", m.Attachments)
	}
	att := m.Attachments[0]
	if att.AttachmentID != "att1" || att.Filename != "facture.pdf" || att.SizeOctets != 1234 {
		t.Errorf("Attachments[0] = %+v", att)
	}
}

func TestParseMessageFromNonParsable(t *testing.T) {
	// Une valeur de "From" mal formée ne doit pas faire échouer le parsing :
	// on garde la valeur brute plutôt que d'abandonner tout le message.
	raw := gmailMessage{
		ID: "msg3",
		Payload: gmailPart{
			Headers: []gmailHeader{{Name: "From", Value: "pas-une-adresse-valide"}},
		},
	}
	m, err := parseMessage(raw)
	if err != nil {
		t.Fatalf("parseMessage: %v", err)
	}
	if m.From != "pas-une-adresse-valide" {
		t.Errorf("From = %q, attendu la valeur brute conservée", m.From)
	}
}
