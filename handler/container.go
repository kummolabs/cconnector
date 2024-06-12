package handler

import (
	"net/http"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
)

// Requests

type ContainerCreationRequest struct {
}

// Handler

type Container struct {
	dockerClient *client.Client
}

func NewContainer(dockerClient *client.Client) *Container {
	return &Container{dockerClient: dockerClient}
}

func (c *Container) List(echoContext echo.Context) error {
	containers, err := c.dockerClient.ContainerList(echoContext.Request().Context(), container.ListOptions{})
	if err != nil {
		return echoContext.JSON(http.StatusInternalServerError, map[any]any{
			"message": "internal server error",
		})
	}

	// TODO: use json api standard
	return echoContext.JSON(http.StatusOK, containers)
}
