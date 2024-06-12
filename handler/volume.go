package handler

import (
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
)

type VolumeCreationRequest struct {
	Labels Label `json:"labels"`
}

type Volume struct {
	dockerClient *client.Client
}

func NewVolume(dockerClient *client.Client) *Volume {
	return &Volume{dockerClient: dockerClient}
}

func (v *Volume) List(c echo.Context) error {
	volumes, err := v.dockerClient.VolumeList(c.Request().Context(), volume.ListOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	// TODO: use json api standard
	return c.JSON(http.StatusOK, volumes)
}

func (v *Volume) Create(c echo.Context) error {
	createOptions := volume.CreateOptions{}

	err := json.NewDecoder(c.Request().Body).Decode(&createOptions)
	if err != nil {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
	}
	vol, err := v.dockerClient.VolumeCreate(c.Request().Context(), createOptions)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	// TODO: use json api standard
	return c.JSON(http.StatusOK, vol)
}
