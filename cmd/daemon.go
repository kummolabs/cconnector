package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/insomnius/agent/entity"
	"github.com/insomnius/agent/handler"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
)

type Daemon struct {
	configPath string
}

func NewDaemon(configPath string) *Daemon {
	return &Daemon{
		configPath: configPath,
	}
}

func (d *Daemon) Start() *cobra.Command {
	return &cobra.Command{
		Use:     "daemon:start",
		Short:   "Run cconector daemon",
		Long:    "Run cconector daemon, which include: HTTP servers",
		GroupID: "daemon",
		Run: func(cmd *cobra.Command, args []string) {
			if err := checkDockerSocketRunning(); err != nil {
				fmt.Printf("Error starting daemon, docker socket is not running. Error:\n%v\n", err)
				return
			}

			currentConfig, err := getConfig(d.configPath)
			if err != nil {
				fmt.Printf("Failed to identify cconector config, you can initiate your config by running `cconector config:initiate`. Errors:\n%v\n", err)
				return
			}

			e := echo.New()

			e.Use(
				middleware.Logger(),
				middleware.Recover(),
			)

			e.GET("/status", func(c echo.Context) error {
				return c.String(http.StatusOK, "OK\n")
			})

			withAuthEngine := e.Group("/v1",
				middleware.KeyAuth(func(auth string, c echo.Context) (bool, error) {
					return auth == currentConfig.HostToken, nil
				}),
			)

			withAuthEngine.GET("/authentication-status", func(c echo.Context) error {
				return c.String(http.StatusOK, "OK\n")
			})

			// Initiate docker clients
			cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				fmt.Println("Failed to create Docker client:", err)
				return
			}

			editConfigWrapper := func(newConfig entity.CconnectorConfig) error {
				return editConfig(d.configPath, newConfig)
			}

			getConfigWrapper := func() (*entity.CconnectorConfig, error) {
				return getConfig(d.configPath)
			}

			managerHandler := handler.NewManager(editConfigWrapper, getConfigWrapper)
			withAuthEngine.POST("/managers/claims", managerHandler.Claim)

			networkHandler := handler.NewNetwork(cli)
			withAuthEngine.POST("/networks", networkHandler.Create)

			containerHandler := handler.NewContainer(cli)
			withAuthEngine.GET("/containers", containerHandler.List)
			withAuthEngine.POST("/containers", containerHandler.Create)

			volumeHandler := handler.NewVolume(cli)
			withAuthEngine.GET("/volumes", volumeHandler.List)
			withAuthEngine.POST("/volumes", volumeHandler.Create)

			imageHandler := handler.NewImage(cli)
			withAuthEngine.POST("/images", imageHandler.Create)

			// Start the server in a goroutine
			go func() {
				port := "30000"
				if os.Getenv("PORT") != "" {
					port = os.Getenv("PORT")
				}

				if err := e.Start(fmt.Sprintf(":%s", port)); err != nil && err != http.ErrServerClosed {
					fmt.Println("Error shutting down the server. Error:", err)
				}
			}()

			// Wait for interrupt signal to gracefully shut down the server with a timeout of 10 seconds.
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
			<-quit
			fmt.Printf("\nShutting down the server...\n")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := e.Shutdown(ctx); err != nil {
				fmt.Println("Error shutting down the server. Error:", err)
			}
		},
	}
}
