package handler

import (
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type VolumeCreationRequest struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	DriverOpts map[string]string `json:"driver_opts"`
	Labels     Label             `json:"labels"`
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
	creationRequest := VolumeCreationRequest{}

	err := json.NewDecoder(c.Request().Body).Decode(&creationRequest)
	if err != nil {
		_ = c.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
		return err
	}

	createOptions := volume.CreateOptions{
		Name:       creationRequest.Name,
		Driver:     creationRequest.Driver,
		DriverOpts: creationRequest.DriverOpts,
		Labels:     creationRequest.Labels,
	}

	vol, err := v.dockerClient.VolumeCreate(c.Request().Context(), createOptions)
	if err != nil {
		log.Err(err).
			Any("create_options", createOptions).
			Msg("error creating volume")

		_ = c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
		return err
	}

	volInspect, err := v.dockerClient.VolumeInspect(c.Request().Context(), vol.Name)
	if err != nil {
		log.Err(err).
			Any("create_options", createOptions).
			Any("volume", vol).
			Msg("error get volume inspect")

		c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())

		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": volInspect,
	})
}
