package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/config"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/metrics"
	"github.com/Abhi1264/unicore/api/internal/pdf"
	"github.com/Abhi1264/unicore/api/internal/queue"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxOutboxAttempts = 8
	outboxBatchSize   = 50

	maxCSVRows      = 20000
	maxCSVBytes     = 16 << 20
	maxCSVFieldSize = 4096
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.AssertRLSEnforced(ctx); err != nil {
		if !cfg.AllowRLSBypass {
			log.Error("refusing to start", "error", err)
			os.Exit(1)
		}
		log.Warn("starting with row-level security bypassed (ALLOW_RLS_BYPASS=true)")
	}

	natsClient, err := queue.New(cfg.NATSURL, log)
	if err != nil {
		log.Error("nats", "error", err)
		os.Exit(1)
	}
	defer natsClient.Close()

	if err := os.MkdirAll(cfg.StoragePath, 0o750); err != nil {
		log.Error("storage path", "error", err)
		os.Exit(1)
	}

	w := &worker{
		pool:        pool,
		nats:        natsClient,
		log:         log,
		storagePath: cfg.StoragePath,
	}

	if natsClient.Available() {
		for _, sub := range []struct {
			topic   string
			durable string
			handler func([]byte) error
		}{
			{queue.TopicPaymentConfirmed, "worker-payments", w.handlePaymentConfirmed},
			{queue.TopicPDFGenerate, "worker-documents", w.handleDocumentGenerate},
			{queue.TopicBulkImport, "worker-bulk", w.handleBulkImport},
		} {
			if err := natsClient.Subscribe(sub.topic, sub.durable, sub.handler); err != nil {
				log.Error("subscribe failed", "topic", sub.topic, "error", err)
				os.Exit(1)
			}
		}
	} else {
		log.Warn("nats unavailable; worker will only poll outbox when broker recovers")
	}

	go w.pollOutbox(ctx)

	log.Info("worker started")
	<-ctx.Done()
	log.Info("worker shutting down")
}

type worker struct {
	pool        *db.Pool
	nats        *queue.Client
	log         *slog.Logger
	storagePath string
}

func (w *worker) pollOutbox(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.drainOutbox(ctx)
		}
	}
}

// drainOutbox relays outbox events; failures are recorded or dead-lettered.
func (w *worker) drainOutbox(ctx context.Context) {
	if !w.nats.Available() {
		return
	}
	events, err := w.pool.ClaimPendingOutbox(ctx, outboxBatchSize, maxOutboxAttempts)
	if err != nil {
		w.log.Error("claim outbox", "error", err)
		return
	}
	for _, e := range events {
		if err := w.nats.Publish(ctx, e.Topic, json.RawMessage(e.Payload)); err != nil {
			w.log.Error("publish outbox event", "topic", e.Topic, "event_id", e.ID, "attempt", e.Attempts, "error", err)
			if markErr := w.pool.MarkOutboxFailed(ctx, e.ID, err.Error(), maxOutboxAttempts); markErr != nil {
				w.log.Error("mark outbox failed", "event_id", e.ID, "error", markErr)
			}
			continue
		}
		if err := w.pool.MarkOutboxPublished(ctx, e.ID); err != nil {
			// Already published; unmarked rows will redeliver (consumers must tolerate).
			w.log.Error("mark outbox published", "event_id", e.ID, "error", err)
		}
	}

	if dead, err := w.pool.CountOutboxDeadLetters(ctx); err == nil {
		metrics.OutboxDeadLetters.Set(float64(dead))
		if dead > 0 {
			w.log.Warn("outbox dead letters awaiting operator attention", "count", dead)
		}
	}
}

func (w *worker) handlePaymentConfirmed(msg []byte) error {
	var p struct {
		PaymentID uuid.UUID `json:"payment_id"`
		TenantID  uuid.UUID `json:"tenant_id"`
	}
	if err := json.Unmarshal(msg, &p); err != nil {
		return err
	}
	w.log.Info("payment confirmed", "payment_id", p.PaymentID, "tenant_id", p.TenantID)
	return nil
}

var documentTypes = map[string]struct{}{
	"marksheet":   {},
	"bonafide":    {},
	"fee_receipt": {},
}

func (w *worker) handleDocumentGenerate(msg []byte) error {
	var p struct {
		Type      string    `json:"type"`
		PaymentID uuid.UUID `json:"payment_id"`
		StudentID uuid.UUID `json:"student_id"`
		TenantID  uuid.UUID `json:"tenant_id"`
		DocID     uuid.UUID `json:"document_id"`
	}
	if err := json.Unmarshal(msg, &p); err != nil {
		return err
	}
	if p.TenantID == uuid.Nil {
		return errors.New("document event missing tenant_id")
	}
	// Type becomes part of a filename; allow-list only.
	if _, ok := documentTypes[p.Type]; !ok {
		return fmt.Errorf("unsupported document type %q", p.Type)
	}

	ctx := context.Background()
	dir := filepath.Join(w.storagePath, p.TenantID.String())
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	var pdfBytes []byte
	var err error
	if p.Type == "fee_receipt" && p.PaymentID != uuid.Nil {
		pdfBytes, err = pdf.GenerateReceipt(pdf.ReceiptInput{
			PaymentID: p.PaymentID.String(),
			PaidAt:    time.Now().UTC(),
		})
	} else {
		pdfBytes, err = pdf.WriteSimplePDF([]string{
			"Unicore Document",
			"Type: " + p.Type,
			"Student: " + p.StudentID.String(),
		})
	}
	if err != nil {
		return err
	}

	name := p.Type + "-" + uuid.NewString() + ".pdf"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, pdfBytes, 0o640); err != nil {
		return err
	}
	rel, err := filepath.Rel(w.storagePath, path)
	if err != nil {
		return err
	}

	return w.pool.WithTenant(ctx, p.TenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		docID := p.DocID
		if docID == uuid.Nil {
			doc, err := q.CreateDocument(ctx, sqlcdb.CreateDocumentParams{
				TenantID:  p.TenantID,
				StudentID: p.StudentID,
				Type:      p.Type,
			})
			if err != nil {
				return err
			}
			docID = doc.ID
		}
		_, err := q.UpdateDocumentStatus(ctx, sqlcdb.UpdateDocumentStatusParams{
			TenantID:   p.TenantID,
			ID:         docID,
			Status:     "ready",
			StorageRef: services.Text(rel),
		})
		return err
	})
}

func (w *worker) handleBulkImport(msg []byte) error {
	var p struct {
		JobID    uuid.UUID `json:"job_id"`
		TenantID uuid.UUID `json:"tenant_id"`
		JobType  string    `json:"job_type"`
	}
	if err := json.Unmarshal(msg, &p); err != nil {
		return err
	}
	if p.TenantID == uuid.Nil || p.JobID == uuid.Nil {
		return errors.New("bulk import event missing ids")
	}
	if p.JobType != "students" && p.JobType != "results" {
		return w.failBulk(context.Background(), p.TenantID, p.JobID, "unsupported job_type")
	}

	ctx := context.Background()
	// Path from uuids only — never the uploader's filename.
	csvPath := filepath.Join(w.storagePath, "bulk", p.TenantID.String(), p.JobID.String()+".csv")
	f, err := os.Open(csvPath)
	if err != nil {
		_ = w.failBulk(ctx, p.TenantID, p.JobID, "upload not found")
		return err
	}
	defer f.Close()

	if info, err := f.Stat(); err == nil && info.Size() > maxCSVBytes {
		_ = w.failBulk(ctx, p.TenantID, p.JobID, "upload exceeds size limit")
		return errors.New("bulk import file too large")
	}

	r := csv.NewReader(io.LimitReader(f, maxCSVBytes))
	r.ReuseRecord = true
	header, err := r.Read()
	if err != nil {
		_ = w.failBulk(ctx, p.TenantID, p.JobID, "csv has no header row")
		return err
	}
	cols := make(map[string]int, len(header))
	for i, h := range header {
		cols[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var total, okCount int32
	errs := make([]string, 0, 16)

	err = w.pool.WithTenant(ctx, p.TenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		for {
			row, err := r.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				errs = appendCapped(errs, fmt.Sprintf("row %d: malformed", total+1))
				continue
			}
			total++
			if total > maxCSVRows {
				errs = appendCapped(errs, fmt.Sprintf("stopped after %d rows", maxCSVRows))
				break
			}

			var rowErr error
			switch p.JobType {
			case "students":
				rowErr = importStudentRow(ctx, q, p.TenantID, cols, row)
			case "results":
				rowErr = importResultRow(ctx, q, p.TenantID, cols, row)
			}
			if rowErr != nil {
				errs = appendCapped(errs, fmt.Sprintf("row %d: %s", total, rowErr))
				continue
			}
			okCount++
		}
		return nil
	})
	if err != nil {
		_ = w.failBulk(ctx, p.TenantID, p.JobID, "import transaction failed")
		return err
	}

	status := "completed"
	if okCount == 0 && len(errs) > 0 {
		status = "failed"
	}
	report, _ := json.Marshal(map[string]any{"errors": errs})
	return w.pool.WithTenant(ctx, p.TenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		_, err := q.UpdateBulkJob(ctx, sqlcdb.UpdateBulkJobParams{
			TenantID:    p.TenantID,
			ID:          p.JobID,
			Status:      status,
			TotalRows:   total,
			SuccessRows: okCount,
			ErrorReport: report,
		})
		return err
	})
}

func appendCapped(errs []string, msg string) []string {
	const maxReported = 100
	if len(errs) >= maxReported {
		return errs
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return append(errs, msg)
}

func (w *worker) failBulk(ctx context.Context, tenantID, jobID uuid.UUID, msg string) error {
	report, _ := json.Marshal(map[string]any{"errors": []string{msg}})
	return w.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		_, err := q.UpdateBulkJob(ctx, sqlcdb.UpdateBulkJobParams{
			TenantID:    tenantID,
			ID:          jobID,
			Status:      "failed",
			TotalRows:   0,
			SuccessRows: 0,
			ErrorReport: report,
		})
		return err
	})
}

func csvGetter(cols map[string]int, row []string) func(string) string {
	return func(k string) string {
		i, ok := cols[k]
		if !ok || i >= len(row) {
			return ""
		}
		v := strings.TrimSpace(row[i])
		if len(v) > maxCSVFieldSize {
			return v[:maxCSVFieldSize]
		}
		return v
	}
}

func importStudentRow(ctx context.Context, q *sqlcdb.Queries, tenantID uuid.UUID, cols map[string]int, row []string) error {
	get := csvGetter(cols, row)
	email := strings.ToLower(get("email"))
	roll := get("roll_number")
	if email == "" || roll == "" {
		return errors.New("email and roll_number are required")
	}

	// Random per-row password if omitted (never a shared default).
	password := get("password")
	if password == "" {
		password = uuid.NewString() + uuid.NewString()
	}
	if err := auth.ValidatePassword(password); err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	user, err := q.CreateUser(ctx, sqlcdb.CreateUserParams{
		TenantID: tenantID, Email: email, PasswordHash: hash,
		Role: string(auth.RoleStudent), FullName: get("full_name"),
	})
	if err != nil {
		return errors.New("could not create user (duplicate email?)")
	}

	batch := int32(time.Now().Year())
	if v := get("batch_year"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1900 || n > 2200 {
			return errors.New("batch_year must be a four digit year")
		}
		batch = int32(n)
	}
	if _, err := q.CreateStudent(ctx, sqlcdb.CreateStudentParams{
		TenantID: tenantID, UserID: user.ID, RollNumber: roll,
		Program: get("program"), BatchYear: batch, DepartmentID: pgtype.UUID{},
	}); err != nil {
		return errors.New("could not create student (duplicate roll number?)")
	}
	return nil
}

func importResultRow(ctx context.Context, q *sqlcdb.Queries, tenantID uuid.UUID, cols map[string]int, row []string) error {
	get := csvGetter(cols, row)
	studentID, err := uuid.Parse(get("student_id"))
	if err != nil {
		return errors.New("student_id is not a valid uuid")
	}
	courseID, err := uuid.Parse(get("course_id"))
	if err != nil {
		return errors.New("course_id is not a valid uuid")
	}
	semester := get("semester")
	grade := get("grade")
	if semester == "" || grade == "" {
		return errors.New("semester and grade are required")
	}

	gp, err := optionalFloat(get("grade_points"), 0, 10)
	if err != nil {
		return fmt.Errorf("grade_points: %w", err)
	}
	marks, err := optionalFloat(get("marks"), 0, 1000)
	if err != nil {
		return fmt.Errorf("marks: %w", err)
	}

	if _, err := q.UpsertResult(ctx, sqlcdb.UpsertResultParams{
		TenantID: tenantID, StudentID: studentID, CourseID: courseID,
		Semester: semester, Grade: grade,
		GradePoints:      services.NumericFromFloatPtr(gp),
		Marks:            services.NumericFromFloatPtr(marks),
		SubmissionStatus: "draft",
		EnteredBy:        pgtype.UUID{},
	}); err != nil {
		return errors.New("could not save result (unknown student or course?)")
	}
	return nil
}

func optionalFloat(v string, min, max float64) (*float64, error) {
	if v == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil, errors.New("not a number")
	}
	if f < min || f > max {
		return nil, fmt.Errorf("must be between %g and %g", min, max)
	}
	return &f, nil
}

