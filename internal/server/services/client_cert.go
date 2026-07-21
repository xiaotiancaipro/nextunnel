package services

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel/internal/server/clients"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	sharedcerts "github.com/xiaotiancaipro/nextunnel/internal/shared/certs"
	"gorm.io/gorm"
)

type ClientCert struct {
	Config   *configs.Cert
	Database *clients.Database
}

type ClientCertView struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt *time.Time
	Serial    string
}

func (s *ClientCert) List(clientID uuid.UUID) ([]ClientCertView, error) {
	var records []models.ClientCert
	if err := s.Database.DB.Where("client_id = ? AND is_delete = ?", clientID, false).
		Order("created_at ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query client certificates: %w", err)
	}

	items := make([]ClientCertView, 0, len(records))
	for i := range records {
		view, err := s.toView(records[i])
		if err != nil {
			return nil, err
		}
		items = append(items, view)
	}
	return items, nil
}

func (s *ClientCert) Create(client *models.Client, expiresAt *time.Time) (ClientCertView, error) {
	notAfter, err := sharedcerts.ResolveNotAfter(expiresAt)
	if err != nil {
		return ClientCertView{}, err
	}

	certID := uuid.New()
	relPath := sharedcerts.RelClientCertPath(client.Name, certID.String())
	absPath, err := sharedcerts.AbsCertPath(s.Config.Dir, relPath)
	if err != nil {
		return ClientCertView{}, err
	}

	certPEM, keyPEM, err := sharedcerts.GenerateClientPEM(s.Config.Dir, s.Config.Host, expiresAt)
	if err != nil {
		return ClientCertView{}, err
	}
	if err := sharedcerts.WriteClientPEMToDir(absPath, certPEM, keyPEM); err != nil {
		return ClientCertView{}, err
	}

	record := models.ClientCert{
		Id:        certID,
		ClientId:  client.Id,
		CertPath:  relPath,
		ExpiredAt: notAfter.UTC(),
	}
	if err := s.Database.DB.Create(&record).Error; err != nil {
		_ = sharedcerts.RemoveCertDir(absPath)
		return ClientCertView{}, fmt.Errorf("failed to create client certificate record: %w", err)
	}

	return s.toView(record)
}

func (s *ClientCert) Delete(clientID uuid.UUID, certID uuid.UUID) error {
	var record models.ClientCert
	if err := s.Database.DB.Where("id = ? AND client_id = ? AND is_delete = ?", certID, clientID, false).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("certificate %q not found", certID)
		}
		return fmt.Errorf("failed to query client certificate: %w", err)
	}

	result := s.Database.DB.Model(&models.ClientCert{}).
		Where("id = ? AND client_id = ? AND is_delete = ?", certID, clientID, false).
		Update("is_delete", true)
	if result.Error != nil {
		return fmt.Errorf("failed to delete client certificate: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("certificate %q not found", certID)
	}

	if absPath, err := sharedcerts.AbsCertPath(s.Config.Dir, record.CertPath); err == nil {
		if err := sharedcerts.RemoveCertDir(absPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (s *ClientCert) DeleteAllForClient(clientID uuid.UUID, clientName string) error {
	if err := s.Database.DB.Model(&models.ClientCert{}).
		Where("client_id = ? AND is_delete = ?", clientID, false).
		Update("is_delete", true).Error; err != nil {
		return fmt.Errorf("failed to delete client certificates: %w", err)
	}

	if err := sharedcerts.RemoveClientCertDir(s.Config.Dir, clientName); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *ClientCert) ReadFiles(clientID uuid.UUID, certID uuid.UUID) ([]byte, []byte, error) {
	record, err := s.getActive(clientID, certID)
	if err != nil {
		return nil, nil, err
	}
	absPath, err := sharedcerts.AbsCertPath(s.Config.Dir, record.CertPath)
	if err != nil {
		return nil, nil, err
	}
	return sharedcerts.ReadCertFiles(absPath)
}

// OwnsFingerprint reports whether peerFP matches an active, unexpired certificate
// registered for clientID. Soft-deleted or missing on-disk certs do not match.
func (s *ClientCert) OwnsFingerprint(clientID uuid.UUID, peerFP [sha256.Size]byte) (bool, error) {
	var records []models.ClientCert
	now := time.Now().UTC()
	if err := s.Database.DB.Where("client_id = ? AND is_delete = ? AND expired_at > ?", clientID, false, now).
		Find(&records).Error; err != nil {
		return false, fmt.Errorf("failed to query client certificates: %w", err)
	}

	matched := 0
	for i := range records {
		fp, err := s.fingerprintOf(records[i])
		if err != nil {
			continue
		}
		matched |= subtle.ConstantTimeCompare(fp[:], peerFP[:])
	}
	return matched == 1, nil
}

func (s *ClientCert) fingerprintOf(record models.ClientCert) ([sha256.Size]byte, error) {
	var z [sha256.Size]byte
	absPath, err := sharedcerts.AbsCertPath(s.Config.Dir, record.CertPath)
	if err != nil {
		return z, err
	}
	certPEM, err := os.ReadFile(filepath.Join(absPath, sharedcerts.FileClientCert))
	if err != nil {
		return z, err
	}
	return sharedcerts.CertPEMSHA256(certPEM)
}

func (s *ClientCert) getActive(clientID, certID uuid.UUID) (*models.ClientCert, error) {
	var record models.ClientCert
	if err := s.Database.DB.Where("id = ? AND client_id = ? AND is_delete = ?", certID, clientID, false).
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("certificate %q not found", certID)
		}
		return nil, fmt.Errorf("failed to query client certificate: %w", err)
	}
	return &record, nil
}

func (s *ClientCert) toView(record models.ClientCert) (ClientCertView, error) {
	view := ClientCertView{
		ID:        record.Id.String(),
		CreatedAt: record.CreatedAt.UTC(),
	}
	if !sharedcerts.IsNeverExpires(record.ExpiredAt) {
		expires := record.ExpiredAt.UTC()
		view.ExpiresAt = &expires
	}

	absPath, err := sharedcerts.AbsCertPath(s.Config.Dir, record.CertPath)
	if err != nil {
		return ClientCertView{}, err
	}
	certPEM, err := os.ReadFile(filepath.Join(absPath, sharedcerts.FileClientCert))
	if err != nil {
		if os.IsNotExist(err) {
			return view, nil
		}
		return ClientCertView{}, fmt.Errorf("read certificate file: %w", err)
	}
	serial, err := sharedcerts.ParseSerial(certPEM)
	if err != nil {
		return ClientCertView{}, err
	}
	view.Serial = serial
	return view, nil
}
