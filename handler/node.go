package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Node handler struct for node-related operations
type Node struct{}

// NewNode creates a new Node handler
func NewNode() *Node {
	return &Node{}
}

// Specs returns the system's hardware specifications
func (n *Node) Specs(c echo.Context) error {
	// Get machine specifications using the existing function from common.go
	specs, err := getMachineSpecs()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	// Return the machine specs in the response
	return c.JSON(http.StatusOK, map[string]any{
		"machine_specs": specs,
	})
}
