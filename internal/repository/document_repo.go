package repository

import (
	"context"
	"customer-onboarding-service/internal/domain"
)

type DocumentRepo struct{}

func NewDocumentRepo() *DocumentRepo { return &DocumentRepo{} }

func (r *DocumentRepo) Insert(ctx context.Context, db DBTX, caseID, verificationID, docType, storageRef string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO document_ref (case_id, verification_id, doc_type, storage_ref)
		VALUES ($1, $2, $3, $4)`,
		caseID, verificationID, docType, storageRef)
	return err
}
func (r *DocumentRepo) SetStatusByVerification(ctx context.Context, db DBTX, verificationID string, st domain.ReviewStatus) error {
	_, err := db.Exec(ctx, `
		UPDATE document_ref SET verification_status = $2::review_status
		WHERE verification_id = $1`, verificationID, string(st))
	return err
}
