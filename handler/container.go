package handler

import (
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Requests

type ContainerCreationRequest struct {
	Name        string   `json:"string"`
	ImageSource string   `json:"image_source"`
	ImageTag    string   `json:"image_tag"`
	Labels      Label    `json:"labels"`
	Networks    []string `json:"networks"` // sets of id of networks
	Volumes     []struct {
		Name        string `json:"name"`
		BindingPath string `json:"binding_path"`
	} `json:"volumes"`
}

// Handler

type Container struct {
	dockerClient *client.Client
}

func NewContainer(dockerClient *client.Client) *Container {
	return &Container{dockerClient: dockerClient}
}

func (c *Container) Create(echoContext echo.Context) error {
	creationRequest := ContainerCreationRequest{}

	err := json.NewDecoder(echoContext.Request().Body).Decode(&creationRequest)
	if err != nil {
		_ = echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
		return err
	}

	c.dockerClient.ContainerCreate(
		echoContext.Request().Context(),
		&container.Config{},
		&container.HostConfig{},
		&network.NetworkingConfig{},
		&v1.Platform{},
		creationRequest.Name,
	)
	return nil
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
