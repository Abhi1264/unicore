package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DocumentsService struct {
	pool *db.Pool
}

func NewDocumentsService(pool *db.Pool) *DocumentsService {
	return &DocumentsService{pool: pool}
}

func (s *DocumentsService) RequestDocument(ctx context.Context, tenantID, studentID uuid.UUID, docType string) (sqlcdb.Document, error) {
	switch docType {
	case "marksheet", "bonafide", "fee_receipt":
	default:
		return sqlcdb.Document{}, ErrInvalidInput
	}

	var doc sqlcdb.Document
	err := s.pool.WithTenantTx(ctx, tenantID, func(ctx context.Context, _ pgx.Tx, q *sqlcdb.Queries) error {
		var err error
		doc, err = q.CreateDocument(ctx, sqlcdb.CreateDocumentParams{
			TenantID:  tenantID,
			StudentID: studentID,
			Type:      docType,
		})
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{
			"document_id": doc.ID,
			"tenant_id":   tenantID,
			"student_id":  studentID,
			"type":        docType,
		})
		_, err = q.InsertOutbox(ctx, sqlcdb.InsertOutboxParams{
			TenantID: tenantID,
			Topic:    queue.TopicPDFGenerate,
			Payload:  payload,
		})
		return err
	})
	return doc, fmtErr("request document", err)
}

func (s *DocumentsService) ListDocuments(ctx context.Context, tenantID, studentID uuid.UUID) ([]sqlcdb.Document, error) {
	var out []sqlcdb.Document
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.ListStudentDocuments(ctx, sqlcdb.ListStudentDocumentsParams{
			TenantID:  tenantID,
			StudentID: studentID,
		})
		return err
	})
	return out, fmtErr("list documents", err)
}

func (s *DocumentsService) GetDocument(ctx context.Context, tenantID, documentID uuid.UUID) (sqlcdb.Document, error) {
	var out sqlcdb.Document
	err := s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		out, err = q.GetDocument(ctx, sqlcdb.GetDocumentParams{TenantID: tenantID, ID: documentID})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return out, fmtErr("get document", err)
}
