package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type ImageCreationRequest struct {
	Source string `json:"source"`
	Tag    string `json:"tag"`
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
