package main

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
)

func createUserEndpoint(c *fiber.Ctx) error {

	// get the user data from the request body

	var user User

	if err := json.Unmarshal(c.Body(), &user); err != nil {
		return handleError(err, c)
	}

	user = CreateUser(user)

	return c.JSON(fiber.Map{
		"error": false,
		"data":  user,
	})
}
