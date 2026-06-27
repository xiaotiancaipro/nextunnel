package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xiaotiancaipro/nextunnel-server/internal/models"
	"gorm.io/gorm"
)

type ClientRegistry struct {
	db *gorm.DB
}

func NewClientRegistry(db *gorm.DB) *ClientRegistry {
	return &ClientRegistry{db: db}
}

func (r *ClientRegistry) Create(name string, portStart, portEnd int) (*models.Client, error) {
	name = trimName(name)
	if name == "" {
		return nil, fmt.Errorf("client name cannot be empty")
	}
	if err := validatePortRange(portStart, portEnd); err != nil {
		return nil, err
	}

	client := models.Client{
		Name:      name,
		PortStart: portStart,
		PortEnd:   portEnd,
	}
	if err := r.db.Create(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fmt.Errorf("client %q already exists", name)
		}
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return &client, nil
}

func (r *ClientRegistry) GetByName(name string) (*models.Client, error) {
	name = trimName(name)
	if name == "" {
		return nil, fmt.Errorf("client name cannot be empty")
	}
	var client models.Client
	if err := r.db.Where("name = ? AND is_delete = ?", name, false).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("client %q not found", name)
		}
		return nil, fmt.Errorf("failed to query client: %w", err)
	}
	return &client, nil
}

func ClientPortLimited(client models.Client) bool {
	return client.PortStart > 0 && client.PortEnd > 0
}

func ClientPortAllowed(client models.Client, port int) bool {
	if !ClientPortLimited(client) {
		return true
	}
	return port >= client.PortStart && port <= client.PortEnd
}

func validatePortRange(portStart, portEnd int) error {
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

func trimName(name string) string {
	return strings.TrimSpace(name)
}
