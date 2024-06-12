package handler

import (
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type NetworkCreationRequest struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Scope  string `json:"scope"`
	Labels Label  `json:"labels"`
}

type Network struct {
	dockerClient *client.Client
}

func NewNetwork(dockerClient *client.Client) *Network {
	return &Network{
		dockerClient: dockerClient,
	}
}

func (n *Network) Create(c echo.Context) error {
	createOptions := NetworkCreationRequest{}

	err := json.NewDecoder(c.Request().Body).Decode(&createOptions)
	if err != nil {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
	}

	network, err := n.dockerClient.NetworkCreate(c.Request().Context(), createOptions.Name, types.NetworkCreate{
		Driver: createOptions.Driver,
		Scope:  createOptions.Scope,
		Labels: createOptions.Labels,
	})
	if err != nil {
		log.Err(err).
			Any("create_options", createOptions).
			Msg("error creating network")

		c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())

		return err
	}

	networkInspect, err := n.dockerClient.NetworkInspect(c.Request().Context(), network.ID, types.NetworkInspectOptions{})
	if err != nil {
		log.Err(err).
			Any("create_options", createOptions).
			Any("network", network).
			Msg("error get network inspect")

		c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())

		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": networkInspect,
	})
}
