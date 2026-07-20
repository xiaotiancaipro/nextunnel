package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/xiaotiancaipro/nextunnel/internal/server/models"
	"gorm.io/gorm"
)

type Client struct {
	DB *gorm.DB
}

func (s *Client) Create(name string, portStart, portEnd int) (*models.Client, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("client name cannot be empty")
	}
	if err := s.validatePortRange(portStart, portEnd); err != nil {
		return nil, err
	}

	client := models.Client{
		Name:      name,
		PortStart: portStart,
		PortEnd:   portEnd,
	}
	if err := s.DB.Create(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fmt.Errorf("client %q already exists", name)
		}
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return &client, nil
}

func (s *Client) List() ([]models.Client, error) {
	var clients []models.Client
	if err := s.DB.Where("is_delete = ?", false).Order("created_at ASC").Find(&clients).Error; err != nil {
		return nil, fmt.Errorf("failed to query clients: %w", err)
	}
	return clients, nil
}

func (s *Client) GetByName(name string) (*models.Client, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("client name cannot be empty")
	}
	var client models.Client
	if err := s.DB.Where("name = ? AND is_delete = ?", name, false).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("client %q not found", name)
		}
		return nil, fmt.Errorf("failed to query client: %w", err)
	}
	return &client, nil
}

func (s *Client) Delete(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("client name cannot be empty")
	}
	result := s.DB.Model(&models.Client{}).Where("name = ? AND is_delete = ?", name, false).Update("is_delete", true)
	if result.Error != nil {
		return fmt.Errorf("failed to delete client: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("client %q not found", name)
	}
	return nil
}

func (s *Client) ResolveClientId(db *gorm.DB, id string) (uuid.UUID, error) {
	var client models.Client
	if uid, err := uuid.Parse(id); err == nil {
		if err := db.Where("id = ? AND is_delete = ?", uid, false).First(&client).Error; err != nil {
			return uuid.Nil, fmt.Errorf("client not found: %w", err)
		}
		return client.Id, nil
	}
	if err := db.Where("name = ? AND is_delete = ?", id, false).First(&client).Error; err != nil {
		return uuid.Nil, fmt.Errorf("client not found: %w", err)
	}
	return client.Id, nil
}

func (s *Client) ClientPortAllowed(client models.Client, port int) bool {
	if !(client.PortStart > 0 && client.PortEnd > 0) {
		return true
	}
	return port >= client.PortStart && port <= client.PortEnd
}

func (s *Client) validatePortRange(portStart, portEnd int) error {
	if portStart == 0 && portEnd == 0 {
		return nil
	}
	if portStart == 0 || portEnd == 0 {
		return fmt.Errorf("--port-start and --port-end must be specified together")
	}
	if portStart > portEnd {
		return fmt.Errorf("--port-start must be less than or equal to --port-end")
	}
	if portStart < 1 || portEnd > 65535 {
		return fmt.Errorf("port range must be between 1 and 65535")
	}
	return nil
}
