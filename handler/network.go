package handler

import (
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
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

type NetworkConnectRequest struct {
	Container      string                   `json:"container"`
	EndpointConfig network.EndpointSettings `json:"endpoint_config"`
}

type Network struct {
	dockerClient *client.Client
}

func NewNetwork(dockerClient *client.Client) *Network {
	return &Network{
		dockerClient: dockerClient,
	}
}

func (n *Network) List(c echo.Context) error {
	filterStr := c.QueryParam("filters")
	filterArgs := filters.NewArgs()

	if filterStr != "" {
		var filterMap map[string][]string
		if err := json.Unmarshal([]byte(filterStr), &filterMap); err == nil {
			for key, values := range filterMap {
				for _, val := range values {
					filterArgs.Add(key, val)
				}
			}
		}
	}

	options := types.NetworkListOptions{
		Filters: filterArgs,
	}

	networks, err := n.dockerClient.NetworkList(c.Request().Context(), options)
	if err != nil {
		log.Err(err).
			Msg("error listing networks")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": networks,
	})
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

func (n *Network) Inspect(c echo.Context) error {
	networkID := c.Param("id")

	if len(networkID) == 0 {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("network id cannot be empty"))
	}

	options := types.NetworkInspectOptions{
		Verbose: c.QueryParam("verbose") == "true",
	}

	network, err := n.dockerClient.NetworkInspect(c.Request().Context(), networkID, options)
	if err != nil {
		log.Err(err).
			Str("network_id", networkID).
			Msg("error inspecting network")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": network,
	})
}

func (n *Network) Remove(c echo.Context) error {
	networkID := c.Param("id")

	if len(networkID) == 0 {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("network id cannot be empty"))
	}

	err := n.dockerClient.NetworkRemove(c.Request().Context(), networkID)
	if err != nil {
		log.Err(err).
			Str("network_id", networkID).
			Msg("error removing network")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Network removed successfully",
		"id":      networkID,
	})
}

func (n *Network) Connect(c echo.Context) error {
	networkID := c.Param("id")

	if len(networkID) == 0 {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("network id cannot be empty"))
	}

	var connectRequest NetworkConnectRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&connectRequest); err != nil {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
	}

	if connectRequest.Container == "" {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	err := n.dockerClient.NetworkConnect(c.Request().Context(), networkID, connectRequest.Container, &connectRequest.EndpointConfig)
	if err != nil {
		log.Err(err).
			Str("network_id", networkID).
			Str("container_id", connectRequest.Container).
			Msg("error connecting container to network")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Container connected to network successfully",
		"network_id":   networkID,
		"container_id": connectRequest.Container,
	})
}

func (n *Network) Disconnect(c echo.Context) error {
	networkID := c.Param("id")

	if len(networkID) == 0 {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("network id cannot be empty"))
	}

	type DisconnectRequest struct {
		Container string `json:"container"`
		Force     bool   `json:"force"`
	}

	var disconnectRequest DisconnectRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&disconnectRequest); err != nil {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
	}

	if disconnectRequest.Container == "" {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	err := n.dockerClient.NetworkDisconnect(c.Request().Context(), networkID, disconnectRequest.Container, disconnectRequest.Force)
	if err != nil {
		log.Err(err).
			Str("network_id", networkID).
			Str("container_id", disconnectRequest.Container).
			Bool("force", disconnectRequest.Force).
			Msg("error disconnecting container from network")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Container disconnected from network successfully",
		"network_id":   networkID,
		"container_id": disconnectRequest.Container,
	})
}

func (n *Network) Prune(c echo.Context) error {
	filterStr := c.QueryParam("filters")
	filterArgs := filters.NewArgs()

	if filterStr != "" {
		var filterMap map[string][]string
		if err := json.Unmarshal([]byte(filterStr), &filterMap); err == nil {
			for key, values := range filterMap {
				for _, val := range values {
					filterArgs.Add(key, val)
				}
			}
		}
	}

	report, err := n.dockerClient.NetworksPrune(c.Request().Context(), filterArgs)
	if err != nil {
		log.Err(err).Msg("error pruning networks")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"networks_deleted": report.NetworksDeleted,
	})
}
