package interfaces

import (
	"fmt"
	"food-app-fiber/application"
	"food-app-fiber/domain/entity"
	"food-app-fiber/infrastructure/auth"
	"food-app-fiber/interfaces/fileupload"
	"github.com/gofiber/fiber/v2"
	"os"
	"strconv"
	"time"
)

type Food struct {
	foodApp    application.FoodAppInterface
	userApp    application.UserAppInterface
	fileUpload fileupload.UploadFileInterface
	tk         auth.TokenInterface
	rd         auth.AuthInterface
}

//Food constructor
func NewFood(fApp application.FoodAppInterface, uApp application.UserAppInterface, fd fileupload.UploadFileInterface, rd auth.AuthInterface, tk auth.TokenInterface) *Food {
	return &Food{
		foodApp:    fApp,
		userApp:    uApp,
		fileUpload: fd,
		rd:         rd,
		tk:         tk,
	}
}

func (fo *Food) SaveFood(c *fiber.Ctx) error {
	//check is the user is authenticated first
	metadata, err := fo.tk.ExtractTokenMetadata(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  false,
			"message": "unauthorized",
		})
	}
	//lookup the metadata in redis:
	userId, err := fo.rd.FetchAuth(metadata.TokenUuid)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  false,
			"message": "unauthorized",
		})
	}
	title := c.FormValue("title")
	description := c.FormValue("description")
	if fmt.Sprintf("%T", title) != "string" || fmt.Sprintf("%T", description) != "string" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success":  false,
			"message": "Invalid json",
		})
	}
	//We initialize a new food for the purpose of validating: in case the payload is empty or an invalid data type is used
	emptyFood := entity.Food{}
	emptyFood.Title = title
	emptyFood.Description = description
	saveFoodError := emptyFood.Validate("")
	if saveFoodError != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success":  false,
			"message": saveFoodError,
		})
	}
	file, err := c.FormFile("food_image")
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success":  false,
			"message": "a valid file is required",
		})
	}
	//check if the user exist
	_, err = fo.userApp.GetUser(userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success":  false,
			"message": "user not found, unauthorized",
		})
	}
	uploadedFile, err := fo.fileUpload.UploadFile(file)
	if err != nil {
		fmt.Println("IT IS HERE")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success":  false,
			"message": err.Error(),
		})
	}
	var food = entity.Food{}
	food.UserID = userId
	food.Title = title
	food.Description = description
	food.FoodImage = uploadedFile
	savedFood, saveErr := fo.foodApp.SaveFood(&food)
	if saveErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":  false,
			"message": saveErr,
		})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":  true,
		"data": savedFood,
	})
}

func (fo *Food) UpdateFood(c *fiber.Ctx) error {
	//Check if the user is authenticated first
	metadata, err := fo.tk.ExtractTokenMetadata(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  false,
			"message": "Unauthorized",
		})
	}
	//lookup the metadata in redis:
	userId, err := fo.rd.FetchAuth(metadata.TokenUuid)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  false,
			"message": "Unauthorized",
		})
	}
	foodId, err := strconv.ParseUint(c.Params("food_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success":  false,
			"message": "invalid request",
		})
	}
	//Since it is a multipart form data we sent, we will do a manual check on each item
	title := c.FormValue("title")
	description := c.FormValue("description")
	if fmt.Sprintf("%T", title) != "string" || fmt.Sprintf("%T", description) != "string" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success":  false,
			"message": "invalid json",
		})
	}
	//We initialize a new food for the purpose of validating: in case the payload is empty or an invalid data type is used
	emptyFood := entity.Food{}
	emptyFood.Title = title
	emptyFood.Description = description
	updateFoodError := emptyFood.Validate("update")
	if updateFoodError != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"success":  false,
			"message": updateFoodError,
		})
	}
	user, err := fo.userApp.GetUser(userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success":  false,
			"message": "user not found, unauthorized",
		})
	}

	//check if the food exist:
	food, err := fo.foodApp.GetFood(foodId)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success":  false,
			"message": err.Error(),
		})
	}
	//if the user id doesnt match with the one we have, dont update. This is the case where an authenticated user tries to update someone else post using postman, curl, etc
	if user.ID != food.UserID {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  false,
			"message": "you are not the owner of this food",
		})
	}
	//Since this is an update request,  a new image may or may not be given.
	// If not image is given, an error occurs. We know this that is why we ignored the error and instead check if the file is nil.
	// if not nil, we process the file by calling the "UploadFile" method.
	// if nil, we used the old one whose path is saved in the database
	file, _ := c.FormFile("food_image")
	if file != nil {
		food.FoodImage, err = fo.fileUpload.UploadFile(file)
		//since i am using Digital Ocean(DO) Spaces to save image, i am appending my DO url here. You can comment this line since you may be using Digital Ocean Spaces.
		food.FoodImage = os.Getenv("DO_SPACES_URL") + food.FoodImage
		if err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"success":  false,
				"message": err.Error(),
			})
		}
	}
	//we dont need to update user's id
	food.Title = title
	food.Description = description
	food.UpdatedAt = time.Now()
	updatedFood, dbUpdateErr := fo.foodApp.UpdateFood(food)
	if dbUpdateErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":  false,
			"message": dbUpdateErr,
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":  true,
		"data": updatedFood,
	})
}

func (fo *Food) GetAllFood(c *fiber.Ctx) error {
	allfood, err := fo.foodApp.GetAllFood()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":  false,
			"message": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":  true,
		"data": allfood,
	})
}

func (fo *Food) GetFoodAndCreator(c *fiber.Ctx) error {
	foodId, err := strconv.ParseUint(c.Params("food_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success":  false,
			"message": "invalid request",
		})
	}
	food, err := fo.foodApp.GetFood(foodId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":  false,
			"message": err.Error(),
		})
	}
	user, err := fo.userApp.GetUser(food.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":  false,
			"message": err.Error(),
		})
	}
	foodAndUser := map[string]interface{}{
		"food":    food,
		"creator": user.PublicUser(),
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":  false,
		"data": foodAndUser,
	})
}

func (fo *Food) DeleteFood(c *fiber.Ctx) error {
	metadata, err := fo.tk.ExtractTokenMetadata(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success":  false,
			"message": "Unauthorized",
		})
	}
	foodId, err := strconv.ParseUint(c.Params("food_id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success":  false,
			"message": "invalid request",
		})
	}
	_, err = fo.userApp.GetUser(metadata.UserId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":  false,
			"message": err.Error(),
		})
	}
	err = fo.foodApp.DeleteFood(foodId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success":  false,
			"message": err.Error(),
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":  true,
		"message": "food deleted",
	})
}
