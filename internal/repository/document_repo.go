package repository

import "context"

type DocumentRepo struct{}

func NewDocumentRepo() *DocumentRepo { return &DocumentRepo{} }

func (r *DocumentRepo) Insert(ctx context.Context, db DBTX, caseID, verificationID, docType, storageRef string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO document_ref (case_id, verification_id, doc_type, storage_ref)
		VALUES ($1, $2, $3, $4)`,
		caseID, verificationID, docType, storageRef)
	return err
}
