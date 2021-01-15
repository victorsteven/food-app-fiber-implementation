package interfaces

import (
	"fmt"
	"food-app-fiber/application"
	"food-app-fiber/domain/entity"
	"food-app-fiber/infrastructure/auth"
	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"os"
	"strconv"
)

type Authenticate struct {
	us application.UserAppInterface
	rd auth.AuthInterface
	tk auth.TokenInterface
}

//Authenticate constructor
func NewAuthenticate(uApp application.UserAppInterface, rd auth.AuthInterface, tk auth.TokenInterface) *Authenticate {
	return &Authenticate{
		us: uApp,
		rd: rd,
		tk: tk,
	}
}

func (au *Authenticate) Login(c *fiber.Ctx) error {
	var user *entity.User

	err := c.BodyParser(&user)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success":  false,
			"message": "Cannot parse JSON",
		})
	}
	//validate request:
	validateUser := user.Validate("login")
	if validateUser != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success":  false,
			"message": validateUser,
		})
	}
	u, userErr := au.us.GetUserByEmailAndPassword(user)
	if userErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":  false,
			"message": userErr,
		})
	}
	ts, tErr := au.tk.CreateToken(u.ID)
	if tErr != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success":  false,
			"message": tErr.Error(),
		})
	}
	saveErr := au.rd.CreateAuth(u.ID, ts)
	if saveErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":  false,
			"message": saveErr.Error(),
		})
	}
	userData := make(map[string]interface{})
	userData["access_token"] = ts.AccessToken
	userData["refresh_token"] = ts.RefreshToken
	userData["id"] = u.ID
	userData["first_name"] = u.FirstName
	userData["last_name"] = u.LastName

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":  true,
		"data": userData,
	})
}

func (au *Authenticate) Logout(c *fiber.Ctx) error {
	//check is the user is authenticated first
	metadata, err := au.tk.ExtractTokenMetadata(c)
	if err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":  false,
			"message": "Unauthorized",
		})
	}
	//if the access token exist and it is still valid, then delete both the access token and the refresh token
	deleteErr := au.rd.DeleteTokens(metadata)
	if deleteErr != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  false,
			"message": deleteErr.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":  true,
		"message": "Successfully logged out",
	})
}

//Refresh is the function that uses the refresh_token to generate new pairs of refresh and access tokens.
func (au *Authenticate) Refresh(c *fiber.Ctx) error {
	mapToken := map[string]string{}
	err := c.BodyParser(&mapToken)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success":  false,
			"message": err.Error(),
		})
	}

	refreshToken := mapToken["refresh_token"]

	//verify the token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("REFRESH_SECRET")), nil
	})
	//any error may be due to token expiration
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  false,
			"message": err.Error(),
		})
	}
	//is token valid?
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  false,
			"message": "Unauthorized",
		})
	}
	//Since token is valid, get the uuid:
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		refreshUuid, ok := claims["refresh_uuid"].(string) //convert the interface to string
		if !ok {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"success":  false,
				"message": "Cannot get uuid",
			})
		}
		userId, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
		if err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"success":  false,
				"message": "Error occurred",
			})
		}
		//Delete the previous Refresh Token
		delErr := au.rd.DeleteRefresh(refreshUuid)
		if delErr != nil { //if any goes wrong
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success":  false,
				"message": "Unauthorized",
			})
		}
		//Create new pairs of refresh and access tokens
		ts, createErr := au.tk.CreateToken(userId)
		if createErr != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success":  false,
				"message": createErr.Error(),
			})
		}
		//save the tokens metadata to redis
		saveErr := au.rd.CreateAuth(userId, ts)
		if saveErr != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success":  false,
				"message": saveErr.Error(),
			})
		}
		tokens := map[string]string{
			"access_token":  ts.AccessToken,
			"refresh_token": ts.RefreshToken,
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success":  true,
			"data": tokens,
		})
	} else {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  true,
			"message": "Refresh token expired",
		})
	}
}
