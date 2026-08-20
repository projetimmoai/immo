package repository

import (
	"context"
	"testing"

	"github.com/projetimmoai/immo/internal/domain"
)

func TestInsertLogAndDelete(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	logTypeID, err := c.LogTypeID(ctx, domain.LogTypeCoproprieteNonIdentifiee)
	if err != nil {
		t.Fatalf("LogTypeID: %v", err)
	}

	message := "test d'intégration internal/repository"
	created, err := c.InsertLog(ctx, &domain.Log{LogTypeID: logTypeID, Message: &message})
	if err != nil {
		t.Fatalf("InsertLog: %v", err)
	}
	t.Cleanup(func() {
		if err := c.DeleteLog(context.Background(), created.ID); err != nil {
			t.Errorf("nettoyage DeleteLog id=%d: %v", created.ID, err)
		}
	})

	if created.ID <= 0 {
		t.Fatalf("InsertLog: ID invalide: %d", created.ID)
	}
	if created.LogTypeID != logTypeID {
		t.Errorf("LogTypeID = %d, attendu %d", created.LogTypeID, logTypeID)
	}
	if created.Message == nil || *created.Message != message {
		t.Errorf("Message = %v, attendu %q", created.Message, message)
	}
}
