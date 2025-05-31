package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type ImageCreationRequest struct {
	Source string `json:"source"`
	Tag    string `json:"tag"`
}

type ImagePullRequest struct {
	Reference string              `json:"reference"`
	Platform  string              `json:"platform,omitempty"`
	Auth      registry.AuthConfig `json:"auth,omitempty"`
}

type ImageTagRequest struct {
	TargetRef string `json:"target_ref"`
}

type Image struct {
	dockerClient *client.Client
}

func NewImage(dockerClient *client.Client) *Image {
	return &Image{
		dockerClient: dockerClient,
	}
}

func (i *Image) Create(c echo.Context) error {
	createOptions := ImageCreationRequest{}

	err := json.NewDecoder(c.Request().Body).Decode(&createOptions)
	if err != nil {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
	}

	refFormat := fmt.Sprintf("%s:%s", createOptions.Source, createOptions.Tag)
	rc, err := i.dockerClient.ImageCreate(c.Request().Context(), refFormat, image.CreateOptions{})
	if err != nil {
		log.Err(err).
			Any("create_options", createOptions).
			Any("ref_format", refFormat).
			Msg("error creating image")

		c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())

		return err
	}
	defer rc.Close()

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(http.StatusOK)

	buf := make([]byte, 4096)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			if _, err := c.Response().Write(buf[:n]); err != nil {
				log.Err(err).Msg("error writing chunk")
				return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
			}

			c.Response().Flush()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.
				Err(err).
				Any("create_options", createOptions).
				Any("ref_format", refFormat).
				Msg("error reading from source")
			return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
		}
	}

	return nil
}

func (i *Image) List(c echo.Context) error {
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

	options := image.ListOptions{
		All:     c.QueryParam("all") == "true",
		Filters: filterArgs,
	}

	images, err := i.dockerClient.ImageList(c.Request().Context(), options)
	if err != nil {
		log.Err(err).Msg("error listing images")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": images,
	})
}

func (i *Image) Pull(c echo.Context) error {
	var pullRequest ImagePullRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&pullRequest); err != nil {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
	}

	if pullRequest.Reference == "" {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("image reference cannot be empty"))
	}

	options := image.PullOptions{
		Platform: pullRequest.Platform,
	}

	// Add authentication if provided
	if pullRequest.Auth.Username != "" || pullRequest.Auth.Password != "" {
		encodedAuth, err := json.Marshal(pullRequest.Auth)
		if err != nil {
			log.Err(err).Msg("error encoding auth config")
			return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
		}
		options.RegistryAuth = string(encodedAuth)
	}

	rc, err := i.dockerClient.ImagePull(c.Request().Context(), pullRequest.Reference, options)
	if err != nil {
		log.Err(err).
			Str("reference", pullRequest.Reference).
			Msg("error pulling image")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}
	defer rc.Close()

	// Stream the pull output to the client
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(http.StatusOK)

	buf := make([]byte, 4096)
	for {
		n, err := rc.Read(buf)
		if n > 0 {
			if _, err := c.Response().Write(buf[:n]); err != nil {
				log.Err(err).Msg("error writing chunk")
				return nil
			}
			c.Response().Flush()
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Err(err).Msg("error reading pull response")
			return nil
		}
	}

	return nil
}

func (i *Image) Inspect(c echo.Context) error {
	imageID := c.Param("id")

	if len(imageID) == 0 {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("image id or name cannot be empty"))
	}

	imageInspect, _, err := i.dockerClient.ImageInspectWithRaw(c.Request().Context(), imageID)
	if err != nil {
		log.Err(err).
			Str("image_id", imageID).
			Msg("error inspecting image")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": imageInspect,
	})
}

func (i *Image) Remove(c echo.Context) error {
	imageID := c.Param("id")

	if len(imageID) == 0 {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("image id or name cannot be empty"))
	}

	options := image.RemoveOptions{
		Force:         c.QueryParam("force") == "true",
		PruneChildren: c.QueryParam("prune") == "true",
	}

	items, err := i.dockerClient.ImageRemove(c.Request().Context(), imageID, options)
	if err != nil {
		log.Err(err).
			Str("image_id", imageID).
			Msg("error removing image")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Image removed successfully",
		"deleted": items,
	})
}

func (i *Image) Tag(c echo.Context) error {
	imageID := c.Param("id")

	if len(imageID) == 0 {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("image id or name cannot be empty"))
	}

	var tagRequest ImageTagRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&tagRequest); err != nil {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("body contains invalid json format"))
	}

	if tagRequest.TargetRef == "" {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("target reference cannot be empty"))
	}

	if err := i.dockerClient.ImageTag(c.Request().Context(), imageID, tagRequest.TargetRef); err != nil {
		log.Err(err).
			Str("image_id", imageID).
			Str("target_ref", tagRequest.TargetRef).
			Msg("error tagging image")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Image tagged successfully",
		"source":  imageID,
		"target":  tagRequest.TargetRef,
	})
}

func (i *Image) Prune(c echo.Context) error {
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

	report, err := i.dockerClient.ImagesPrune(c.Request().Context(), filterArgs)
	if err != nil {
		log.Err(err).Msg("error pruning images")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"images_deleted":  report.ImagesDeleted,
		"space_reclaimed": report.SpaceReclaimed,
	})
}

func (i *Image) History(c echo.Context) error {
	imageID := c.Param("id")

	if len(imageID) == 0 {
		return c.JSON(http.StatusBadRequest, BadRequestResponseBody("image id or name cannot be empty"))
	}

	history, err := i.dockerClient.ImageHistory(c.Request().Context(), imageID)
	if err != nil {
		log.Err(err).
			Str("image_id", imageID).
			Msg("error getting image history")
		return c.JSON(http.StatusInternalServerError, InternalServerErrorResponseBody())
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": history,
	})
}
