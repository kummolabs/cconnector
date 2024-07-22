package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/labstack/echo/v4"
	v1 "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Requests

type ContainerCreationRequest struct {
	Name        string   `json:"name"`
	ImageSource string   `json:"image_source"`
	ImageTag    string   `json:"image_tag"`
	Labels      Label    `json:"labels"`
	Networks    []string `json:"networks"` // sets of id of networks
	Volumes     []struct {
		Name        string `json:"name"`
		Destination string `json:"destination"`
	} `json:"volumes"`
	Environments []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"environments"`
	PortBindings []struct {
		Protocol string `json:"protocol"`
		Mapping  []struct {
			HostIP   string `json:"host_ip"`
			HostPort string `json:"host_port"`
		} `json:"mapping"`
	} `json:"port_bindings"`
}

type ContainerStartRequest struct {
	ContainerId string `json:"container_id"`
}

// Handler

type Container struct {
	dockerClient *client.Client
}

func NewContainer(dockerClient *client.Client) *Container {
	return &Container{dockerClient: dockerClient}
}

func (c *Container) Start(echoContext echo.Context) error {
	containerID := echoContext.Param("id")

	if len(containerID) == 0 {
		log.Err(errors.New("container id is empty")).
			Array("tags", zerolog.Arr().Str("container").Str("start").Str("param")).
			Stack().
			Msg("error starting container")
		_ = echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot empty"))
	}

	err := c.dockerClient.ContainerStart(echoContext.Request().Context(), containerID, container.StartOptions{})
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("start").Str("container_start")).
			Stack().
			Msg("error starting container")
		_ = echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
		return err
	}

	containerJson, err := c.dockerClient.ContainerInspect(echoContext.Request().Context(), containerID)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("create").Str("container_inspect")).
			Stack().
			Msg("error inspecting newly created container")
		_ = echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
		return err
	}

	_ = echoContext.JSON(http.StatusOK, containerJson)
	return nil
}

func (c *Container) Create(echoContext echo.Context) error {
	creationRequest := ContainerCreationRequest{}

	err := json.NewDecoder(echoContext.Request().Body).Decode(&creationRequest)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("create").Str("json_decode")).
			Stack().
			Msg("error creating container")
		_ = echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
		return err
	}

	imageRef := fmt.Sprintf("%s:%s", creationRequest.ImageSource, creationRequest.ImageTag)

	envVariables := []string{}
	for _, env := range creationRequest.Environments {
		envVariables = append(envVariables, fmt.Sprintf(`%s="%s"`, env.Key, env.Value))
	}

	volumeBinds := []string{}
	for _, bind := range creationRequest.Volumes {
		volumeBinds = append(volumeBinds, fmt.Sprintf(`%s:%s`, bind.Name, bind.Destination))
	}

	portBindings := nat.PortMap{}
	for _, port := range creationRequest.PortBindings {
		if _, ok := portBindings[nat.Port(port.Protocol)]; !ok {
			portBindings[nat.Port(port.Protocol)] = []nat.PortBinding{}
		}

		for _, binding := range port.Mapping {
			portBindings[nat.Port(port.Protocol)] = append(portBindings[nat.Port(port.Protocol)], nat.PortBinding{
				HostIP:   binding.HostIP,
				HostPort: binding.HostPort,
			})
		}
	}

	networkEndpointConfigs := map[string]*network.EndpointSettings{}
	for _, n := range creationRequest.Networks {
		if _, ok := networkEndpointConfigs[n]; !ok {
			networkResp, err := c.dockerClient.NetworkInspect(echoContext.Request().Context(), n, types.NetworkInspectOptions{Verbose: true})
			if err != nil {
				log.Err(err).
					Array("tags", zerolog.Arr().Str("container").Str("create").Str("network").Str("network_inspect")).
					Stack().
					Msg("error creating container")
				_ = echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
				return err
			}
			networkEndpointConfigs[n] = &network.EndpointSettings{
				NetworkID: networkResp.ID,
				Aliases: []string{
					creationRequest.Name,
				},
			}
		}
	}

	createResp, err := c.dockerClient.ContainerCreate(
		echoContext.Request().Context(),
		&container.Config{
			Image:       imageRef,
			Env:         envVariables,
			Labels:      creationRequest.Labels,
			Healthcheck: &v1.HealthcheckConfig{},
		}, // currently nil, since there are no current usecases
		&container.HostConfig{
			Binds:        volumeBinds,
			PortBindings: portBindings,
		}, // currently nil, since there are no current usecases
		&network.NetworkingConfig{
			EndpointsConfig: networkEndpointConfigs,
		}, // currently nil, since there are no current usecases
		nil, // currently nil, since there are no current usecases
		creationRequest.Name,
	)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("create").Str("container_create")).
			Stack().
			Msg("error creating container")
		_ = echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
		return err
	}

	containerJson, err := c.dockerClient.ContainerInspect(echoContext.Request().Context(), createResp.ID)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("create").Str("container_inspect")).
			Stack().
			Msg("error inspecting newly created container")
		_ = echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
		return err
	}

	_ = echoContext.JSON(http.StatusOK, containerJson)
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
