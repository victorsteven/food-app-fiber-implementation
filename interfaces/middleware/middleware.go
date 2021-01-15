package middleware

import (
	"food-app-fiber/infrastructure/auth"
	"github.com/gofiber/fiber/v2"
)

func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := auth.TokenValid(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success":  false,
				"message": err.Error(),
			})
		}
		return c.Next()
	}
}

