package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel/internal/models"
	"github.com/xiaotiancaipro/nextunnel/internal/utils/certs"
	"gorm.io/gorm"
)

// ClientCertView is the certificate metadata exposed to API/CLI callers.
type ClientCertView struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt *time.Time
	Serial    string
}

type ClientCertRegistry struct {
	db         *gorm.DB
	certDir    string
	listenHost string
}

func NewClientCertRegistry(db *gorm.DB, certDir, listenHost string) *ClientCertRegistry {
	return &ClientCertRegistry{
		db:         db,
		certDir:    certDir,
		listenHost: listenHost,
	}
}

func (r *ClientCertRegistry) List(clientID uuid.UUID) ([]ClientCertView, error) {
	var records []models.ClientCert
	if err := r.db.Where("client_id = ? AND is_delete = ?", clientID, false).
		Order("created_at ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query client certificates: %w", err)
	}

	items := make([]ClientCertView, 0, len(records))
	for i := range records {
		view, err := r.toView(records[i])
		if err != nil {
			return nil, err
		}
		items = append(items, view)
	}
	return items, nil
}

func (r *ClientCertRegistry) Create(client *models.Client, expiresAt *time.Time) (ClientCertView, error) {
	notAfter, err := certs.ResolveNotAfter(expiresAt)
	if err != nil {
		return ClientCertView{}, err
	}

	certID := uuid.New()
	relPath := certs.RelClientCertPath(client.Name, certID.String())
	absPath, err := certs.AbsCertPath(r.certDir, relPath)
	if err != nil {
		return ClientCertView{}, err
	}

	certPEM, keyPEM, err := certs.GenerateClientPEM(r.certDir, r.listenHost, expiresAt)
	if err != nil {
		return ClientCertView{}, err
	}
	if err := certs.WriteClientPEMToDir(absPath, certPEM, keyPEM); err != nil {
		return ClientCertView{}, err
	}

	record := models.ClientCert{
		Id:        certID,
		ClientId:  client.Id,
		CertPath:  relPath,
		ExpiredAt: notAfter.UTC(),
	}
	if err := r.db.Create(&record).Error; err != nil {
		_ = certs.RemoveCertDir(absPath)
		return ClientCertView{}, fmt.Errorf("failed to create client certificate record: %w", err)
	}

	return r.toView(record)
}

func (r *ClientCertRegistry) Delete(clientID uuid.UUID, certID uuid.UUID) error {
	var record models.ClientCert
	if err := r.db.Where("id = ? AND client_id = ? AND is_delete = ?", certID, clientID, false).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("certificate %q not found", certID)
		}
		return fmt.Errorf("failed to query client certificate: %w", err)
	}

	result := r.db.Model(&models.ClientCert{}).
		Where("id = ? AND client_id = ? AND is_delete = ?", certID, clientID, false).
		Update("is_delete", true)
	if result.Error != nil {
		return fmt.Errorf("failed to delete client certificate: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("certificate %q not found", certID)
	}

	if absPath, err := certs.AbsCertPath(r.certDir, record.CertPath); err == nil {
		if err := certs.RemoveCertDir(absPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (r *ClientCertRegistry) DeleteAllForClient(clientID uuid.UUID, clientName string) error {
	if err := r.db.Model(&models.ClientCert{}).
		Where("client_id = ? AND is_delete = ?", clientID, false).
		Update("is_delete", true).Error; err != nil {
		return fmt.Errorf("failed to delete client certificates: %w", err)
	}

	if err := certs.RemoveClientCertDir(r.certDir, clientName); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (r *ClientCertRegistry) ReadFiles(clientID uuid.UUID, certID uuid.UUID) ([]byte, []byte, error) {
	record, err := r.getActive(clientID, certID)
	if err != nil {
		return nil, nil, err
	}
	absPath, err := certs.AbsCertPath(r.certDir, record.CertPath)
	if err != nil {
		return nil, nil, err
	}
	return certs.ReadCertFiles(absPath)
}

func (r *ClientCertRegistry) getActive(clientID, certID uuid.UUID) (*models.ClientCert, error) {
	var record models.ClientCert
	if err := r.db.Where("id = ? AND client_id = ? AND is_delete = ?", certID, clientID, false).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("certificate %q not found", certID)
		}
		return nil, fmt.Errorf("failed to query client certificate: %w", err)
	}
	return &record, nil
}

func (r *ClientCertRegistry) toView(record models.ClientCert) (ClientCertView, error) {
	view := ClientCertView{
		ID:        record.Id.String(),
		CreatedAt: record.CreatedAt.UTC(),
	}
	if !certs.IsNeverExpires(record.ExpiredAt) {
		expires := record.ExpiredAt.UTC()
		view.ExpiresAt = &expires
	}

	absPath, err := certs.AbsCertPath(r.certDir, record.CertPath)
	if err != nil {
		return ClientCertView{}, err
	}
	certPEM, err := os.ReadFile(filepath.Join(absPath, certs.FileClientCert))
	if err != nil {
		if os.IsNotExist(err) {
			return view, nil
		}
		return ClientCertView{}, fmt.Errorf("read certificate file: %w", err)
	}
	serial, err := certs.ParseSerial(certPEM)
	if err != nil {
		return ClientCertView{}, err
	}
	view.Serial = serial
	return view, nil
}

func ParseCertID(raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, fmt.Errorf("certificate id is required")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid certificate id %q", raw)
	}
	return id, nil
}
