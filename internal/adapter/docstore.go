package adapter

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type DocumentStore interface {
	// Put menyimpan file terenkripsi dan mengembalikan storage_ref.
	Put(ctx context.Context, caseID, docType, ext string, data []byte) (string, error)
}

// DummyDocStore: belum menyimpan apa pun, hanya membuat referensi.
// TODO: ganti dengan client Document Management Service.
type DummyDocStore struct{}

func NewDummyDocStore() *DummyDocStore { return &DummyDocStore{} }

func (d *DummyDocStore) Put(_ context.Context, caseID, docType, ext string, _ []byte) (string, error) {
	return fmt.Sprintf("doc-store://%s/%s/%s.%s",
		strings.ToLower(docType), caseID, uuid.NewString(), ext), nil
}
