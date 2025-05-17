package handler

import (
	"encoding/json"
	"net/http"

	"github.com/insomnius/agent/entity"
	"github.com/labstack/echo/v4"
)

// Requests

type ClaimRequest struct {
	ManagerToken string `json:"manager_token"`
}

type Manager struct {
	editConfigFunction func(newConfig entity.CconnectorConfig) error
	getConfigFunction  func() (*entity.CconnectorConfig, error)
}

func NewManager(editConfigFunction func(newConfig entity.CconnectorConfig) error, getConfigFunction func() (*entity.CconnectorConfig, error)) *Manager {
	return &Manager{
		editConfigFunction: editConfigFunction,
		getConfigFunction:  getConfigFunction,
	}
}

func (m *Manager) Claim(c echo.Context) error {
	claimRequest := ClaimRequest{}

	err := json.NewDecoder(c.Request().Body).Decode(&claimRequest)
	if err != nil {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
	}

	defaultConfig, err := m.getConfigFunction()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	if defaultConfig.ManagerToken != "" {
		return c.JSON(http.StatusUnprocessableEntity, UnprocessableEntityResponseBody("manager token already claimed"))
	}

	if err := m.editConfigFunction(entity.CconnectorConfig{
		HostToken:    defaultConfig.HostToken,
		ManagerToken: claimRequest.ManagerToken,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	// Get machine specifications
	specs, err := getMachineSpecs()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message":       "OK",
		"machine_specs": specs,
	})
}
