package handler

import (
	"net/http"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
)

type Volume struct {
	dockerClient *client.Client
}

func NewVolume(dockerClient *client.Client) *Volume {
	return &Volume{dockerClient: dockerClient}
}

func (v *Volume) List(c echo.Context) error {
	containers, err := v.dockerClient.ImageList(c.Request().Context(), image.ListOptions{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[any]any{
			"message": "internal server error",
		})
	}

	return c.JSON(http.StatusOK, containers)
}

// func (v *Volume) Create(c echo.Context) error {
// 	containers, err := v.dockerClient.ImageList(c.Request().Context(), image.ListOptions{})
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[any]any{
// 			"message": "internal server error",
// 		})
// 	}

// 	return c.JSON(http.StatusOK, containers)
// }
