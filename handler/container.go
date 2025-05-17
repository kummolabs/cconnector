package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

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

type ContainerExecRequest struct {
	Cmd          []string `json:"cmd"`
	AttachStdin  bool     `json:"attach_stdin"`
	AttachStdout bool     `json:"attach_stdout"`
	AttachStderr bool     `json:"attach_stderr"`
	Tty          bool     `json:"tty"`
	Detach       bool     `json:"detach"`
	WorkingDir   string   `json:"working_dir,omitempty"`
	Env          []string `json:"env,omitempty"`
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

func (c *Container) Stop(echoContext echo.Context) error {
	containerID := echoContext.Param("id")

	if len(containerID) == 0 {
		log.Err(errors.New("container id is empty")).
			Array("tags", zerolog.Arr().Str("container").Str("stop").Str("param")).
			Stack().
			Msg("error stopping container")
		return echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	// Use a default timeout of 10 seconds
	timeout := int(10 * time.Second)
	err := c.dockerClient.ContainerStop(echoContext.Request().Context(), containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("stop").Str("container_stop")).
			Stack().
			Msg("error stopping container")
		return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return echoContext.JSON(http.StatusOK, map[string]interface{}{
		"message": "Container stopped successfully",
		"id":      containerID,
	})
}

func (c *Container) Remove(echoContext echo.Context) error {
	containerID := echoContext.Param("id")

	if len(containerID) == 0 {
		log.Err(errors.New("container id is empty")).
			Array("tags", zerolog.Arr().Str("container").Str("remove").Str("param")).
			Stack().
			Msg("error removing container")
		return echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	options := container.RemoveOptions{
		RemoveVolumes: echoContext.QueryParam("remove_volumes") == "true",
		RemoveLinks:   echoContext.QueryParam("remove_links") == "true",
		Force:         echoContext.QueryParam("force") == "true",
	}

	err := c.dockerClient.ContainerRemove(echoContext.Request().Context(), containerID, options)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("remove").Str("container_remove")).
			Stack().
			Msg("error removing container")
		return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return echoContext.JSON(http.StatusOK, map[string]interface{}{
		"message": "Container removed successfully",
		"id":      containerID,
	})
}

func (c *Container) Inspect(echoContext echo.Context) error {
	containerID := echoContext.Param("id")

	if len(containerID) == 0 {
		log.Err(errors.New("container id is empty")).
			Array("tags", zerolog.Arr().Str("container").Str("inspect").Str("param")).
			Stack().
			Msg("error inspecting container")
		return echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	containerInfo, err := c.dockerClient.ContainerInspect(echoContext.Request().Context(), containerID)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("inspect").Str("container_inspect")).
			Stack().
			Msg("error inspecting container")
		return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return echoContext.JSON(http.StatusOK, containerInfo)
}

func (c *Container) Pause(echoContext echo.Context) error {
	containerID := echoContext.Param("id")

	if len(containerID) == 0 {
		log.Err(errors.New("container id is empty")).
			Array("tags", zerolog.Arr().Str("container").Str("pause").Str("param")).
			Stack().
			Msg("error pausing container")
		return echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	err := c.dockerClient.ContainerPause(echoContext.Request().Context(), containerID)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("pause").Str("container_pause")).
			Stack().
			Msg("error pausing container")
		return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return echoContext.JSON(http.StatusOK, map[string]interface{}{
		"message": "Container paused successfully",
		"id":      containerID,
	})
}

func (c *Container) Unpause(echoContext echo.Context) error {
	containerID := echoContext.Param("id")

	if len(containerID) == 0 {
		log.Err(errors.New("container id is empty")).
			Array("tags", zerolog.Arr().Str("container").Str("unpause").Str("param")).
			Stack().
			Msg("error unpausing container")
		return echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	err := c.dockerClient.ContainerUnpause(echoContext.Request().Context(), containerID)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("unpause").Str("container_unpause")).
			Stack().
			Msg("error unpausing container")
		return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return echoContext.JSON(http.StatusOK, map[string]interface{}{
		"message": "Container unpaused successfully",
		"id":      containerID,
	})
}

func (c *Container) Logs(echoContext echo.Context) error {
	containerID := echoContext.Param("id")

	if len(containerID) == 0 {
		log.Err(errors.New("container id is empty")).
			Array("tags", zerolog.Arr().Str("container").Str("logs").Str("param")).
			Stack().
			Msg("error getting container logs")
		return echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
		Tail:       echoContext.QueryParam("tail"),
		Timestamps: echoContext.QueryParam("timestamps") == "true",
	}

	logsReader, err := c.dockerClient.ContainerLogs(echoContext.Request().Context(), containerID, options)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("logs").Str("container_logs")).
			Stack().
			Msg("error getting container logs")
		return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}
	defer logsReader.Close()

	// Stream the logs to the response
	echoContext.Response().Header().Set(echo.HeaderContentType, "application/octet-stream")
	echoContext.Response().WriteHeader(http.StatusOK)

	buf := make([]byte, 4096)
	for {
		n, err := logsReader.Read(buf)
		if n > 0 {
			if _, err := echoContext.Response().Write(buf[:n]); err != nil {
				log.Err(err).Msg("error writing logs chunk")
				return nil
			}
			echoContext.Response().Flush()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Err(err).Msg("error reading logs")
			return nil
		}
	}

	return nil
}

func (c *Container) Exec(echoContext echo.Context) error {
	containerID := echoContext.Param("id")
	if len(containerID) == 0 {
		log.Err(errors.New("container id is empty")).
			Array("tags", zerolog.Arr().Str("container").Str("exec").Str("param")).
			Stack().
			Msg("error executing command in container")
		return echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	var execReq ContainerExecRequest
	if err := json.NewDecoder(echoContext.Request().Body).Decode(&execReq); err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("exec").Str("json_decode")).
			Stack().
			Msg("error decoding exec request")
		return echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
	}

	// Create exec instance
	execConfig := types.ExecConfig{
		Cmd:          execReq.Cmd,
		AttachStdin:  execReq.AttachStdin,
		AttachStdout: execReq.AttachStdout,
		AttachStderr: execReq.AttachStderr,
		Tty:          execReq.Tty,
		WorkingDir:   execReq.WorkingDir,
		Env:          execReq.Env,
	}

	execID, err := c.dockerClient.ContainerExecCreate(echoContext.Request().Context(), containerID, execConfig)
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("exec").Str("exec_create")).
			Stack().
			Msg("error creating exec")
		return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	// If detached mode, just return the exec ID
	if execReq.Detach {
		err := c.dockerClient.ContainerExecStart(echoContext.Request().Context(), execID.ID, types.ExecStartCheck{
			Detach: true,
		})
		if err != nil {
			log.Err(err).
				Array("tags", zerolog.Arr().Str("container").Str("exec").Str("exec_start")).
				Stack().
				Msg("error starting exec")
			return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
		}

		return echoContext.JSON(http.StatusOK, map[string]interface{}{
			"message": "Command executed in detached mode",
			"exec_id": execID.ID,
		})
	}

	// For attached mode, stream the output
	resp, err := c.dockerClient.ContainerExecAttach(echoContext.Request().Context(), execID.ID, types.ExecStartCheck{})
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("exec").Str("exec_attach")).
			Stack().
			Msg("error attaching to exec")
		return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}
	defer resp.Close()

	// Stream the exec output to the response
	echoContext.Response().Header().Set(echo.HeaderContentType, "application/octet-stream")
	echoContext.Response().WriteHeader(http.StatusOK)

	buf := make([]byte, 4096)
	for {
		n, err := resp.Reader.Read(buf)
		if n > 0 {
			if _, err := echoContext.Response().Write(buf[:n]); err != nil {
				log.Err(err).Msg("error writing exec output")
				return nil
			}
			echoContext.Response().Flush()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Err(err).Msg("error reading exec output")
			return nil
		}
	}

	return nil
}

func (c *Container) Restart(echoContext echo.Context) error {
	containerID := echoContext.Param("id")

	if len(containerID) == 0 {
		log.Err(errors.New("container id is empty")).
			Array("tags", zerolog.Arr().Str("container").Str("restart").Str("param")).
			Stack().
			Msg("error restarting container")
		return echoContext.JSON(http.StatusBadRequest, BadRequestResponseBody("container id cannot be empty"))
	}

	// Default timeout of 10 seconds
	timeout := int(10 * time.Second)
	err := c.dockerClient.ContainerRestart(echoContext.Request().Context(), containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		log.Err(err).
			Array("tags", zerolog.Arr().Str("container").Str("restart").Str("container_restart")).
			Stack().
			Msg("error restarting container")
		return echoContext.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return echoContext.JSON(http.StatusOK, map[string]interface{}{
		"message": "Container restarted successfully",
		"id":      containerID,
	})
}
