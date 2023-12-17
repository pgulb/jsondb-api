package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Valgard/godotenv"
	"github.com/gin-gonic/gin"
	"github.com/pgulb/jsondb/db"
	"github.com/pgulb/jsondb/structures"
)

func logInfo(c *gin.Context) {
	log.Printf("%s %s\n", c.Request.RequestURI, c.Request.RemoteAddr)
}

func main() {
	timeout := 5
	input := make(chan structures.Request)
	output := make(chan structures.Response)
	initialOutput := make(chan structures.Response)
	cmdArgs := os.Args[1:]

	log.Println("starting jsondb goroutine")
	go db.Listen(cmdArgs, input, output, initialOutput)

	resp, err := db.HandleOutput(initialOutput, timeout)
	for _, v := range resp {
		log.Println(v)
	}
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	input <- structures.Request{
		KeyFamily: "healthcheck",
		Key:       "h",
		Value:     "OK",
		Action:    "set",
	}
	resp, err = db.HandleOutput(output, timeout)
	for _, v := range resp {
		log.Println(v)
	}
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	dotenv := godotenv.New()
	if err := dotenv.Load(".env"); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	r := gin.Default()
	a := r.Group("/", gin.BasicAuth(
		gin.Accounts{
			os.Getenv("API_USER"): os.Getenv("API_PASS"),
		},
	))

	r.GET("/health", func(c *gin.Context) {
		input <- structures.Request{
			KeyFamily: "healthcheck",
			Key:       "h",
			Action:    "get",
		}
		resp, err := db.HandleOutput(output, timeout)
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
		} else {
			c.Data(http.StatusOK, gin.MIMEJSON, []byte(resp[0]))
		}
		logInfo(c)
	})

	r.GET("/values", func(c *gin.Context) {
		input <- structures.Request{
			KeyFamily: "ram_usage",
			Action:    "listkeys",
		}
		resp, err := db.HandleOutput(output, timeout)
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
		} else {
			c.Data(http.StatusOK, gin.MIMEJSON, []byte(resp[0]))
		}
		logInfo(c)
	})

	r.GET("/value/:value", func(c *gin.Context) {
		value := c.Params.ByName("value")
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "empty value was given",
			})
		} else {
			input <- structures.Request{
				Key:       value,
				KeyFamily: "ram_usage",
				Action:    "get",
			}
			resp, err := db.HandleOutput(output, timeout)
			if err != nil {
				log.Fatal(err)
				c.JSON(http.StatusBadRequest, gin.H{
					"message": err.Error(),
				})
			} else {
				c.Data(http.StatusOK, gin.MIMEJSON, []byte(resp[0]))
			}
		}
		logInfo(c)
	})

	r.GET("/latest_value", func(c *gin.Context) {
		input <- structures.Request{
			Key:       "latest",
			KeyFamily: "ram_usage",
			Action:    "get",
		}
		resp, err := db.HandleOutput(output, timeout)
		if err != nil {
			log.Fatal(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
		} else {
			latest := make(map[string]string)
			err := json.Unmarshal([]byte(resp[0]), &latest)
			if err != nil {
				log.Fatal(err)
				c.JSON(http.StatusBadRequest, gin.H{
					"message": err.Error(),
				})
			} else {
				input <- structures.Request{
					Key:       latest["0"],
					KeyFamily: "ram_usage",
					Action:    "get",
				}
				resp, err := db.HandleOutput(output, timeout)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"message": err.Error(),
					})
				} else {
					latest_value := make(map[string]string)
					err := json.Unmarshal([]byte(resp[0]), &latest_value)
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{
							"message": err.Error(),
						})
					} else {
						c.JSON(http.StatusOK, gin.H{
							latest["0"]: latest_value["0"],
						})
					}
				}
			}
		}
		logInfo(c)
	})

	a.POST("input/:value", func(c *gin.Context) {
		now := time.Now().Format("2006-01-02T15:04")
		value := c.Params.ByName("value")
		if value == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "empty value was given",
			})
		} else {
			input <- structures.Request{
				KeyFamily: "ram_usage",
				Key:       now,
				Value:     value,
				Action:    "set",
			}
			_, err := db.HandleOutput(output, timeout)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"message": err.Error(),
				})
			} else {
				input <- structures.Request{
					KeyFamily: "ram_usage",
					Key:       "latest",
					Value:     now,
					Action:    "set",
				}
				resp, err := db.HandleOutput(output, timeout)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"message": err.Error(),
					})
				} else {
					c.JSON(http.StatusOK, gin.H{
						"message": resp,
					})
				}
			}
		}
		logInfo(c)
	})

	r.Run()
}
