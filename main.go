package main

import (
	"food-app-fiber/infrastructure/auth"
	"food-app-fiber/infrastructure/persistence"
	"food-app-fiber/interfaces"
	"food-app-fiber/interfaces/fileupload"
	"food-app-fiber/interfaces/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func init() {
	//To load our environmental variables.
	if err := godotenv.Load(); err != nil {
		log.Println("no env gotten")
	}
}

func main() {

	dbdriver := os.Getenv("DB_DRIVER")
	host := os.Getenv("DB_HOST")
	password := os.Getenv("DB_PASSWORD")
	user := os.Getenv("DB_USER")
	dbname := os.Getenv("DB_NAME")
	port := os.Getenv("DB_PORT")

	//redis details
	redis_host := os.Getenv("REDIS_HOST")
	redis_port := os.Getenv("REDIS_PORT")
	redis_password := os.Getenv("REDIS_PASSWORD")


	services, err := persistence.NewRepositories(dbdriver, user, password, port, host, dbname)
	if err != nil {
		panic(err)
	}
	defer services.Close()
	services.Automigrate()

	redisService, err := auth.NewRedisDB(redis_host, redis_port, redis_password)
	if err != nil {
		log.Fatal(err)
	}

	tk := auth.NewToken()
	fd := fileupload.NewFileUpload()

	users := interfaces.NewUsers(services.User, redisService.Auth, tk)
	foods := interfaces.NewFood(services.Food, services.User, fd, redisService.Auth, tk)
	authenticate := interfaces.NewAuthenticate(services.User, redisService.Auth, tk)

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})

	app.Use(cors.New())

	//user routes
	app.Post("/users", users.SaveUser)
	app.Get("/users", users.GetUsers)
	app.Get("/users/:user_id", users.GetUser)

	//post routes
	app.Post("/food", middleware.AuthMiddleware(), foods.SaveFood)
	app.Put("/food/:food_id", middleware.AuthMiddleware(), foods.UpdateFood)
	app.Get("/food/:food_id", foods.GetFoodAndCreator)
	app.Delete("/food/:food_id", middleware.AuthMiddleware(), foods.DeleteFood)
	app.Get("/food", foods.GetAllFood)

	//authentication routes
	app.Post("/login", authenticate.Login)
	app.Post("/logout", authenticate.Logout)
	app.Post("/refresh", authenticate.Refresh)


	//Starting the application
	app_port := os.Getenv("PORT") //using heroku host
	if app_port == "" {
		app_port = "8888" //localhost
	}

	log.Fatal(app.Listen(":"+app_port))
}

//func Routing() fiber.Route {
//
//	app := fiber.New()
//
//	app.Get("/", func(c *fiber.Ctx) error {
//		return c.SendString("Hello, World 👋!")
//	})
//
//	app.Use(cors.New())
//
//	//user routes
//	app.Post("/users", users.SaveUser)
//	app.Get("/users", users.GetUsers)
//	app.Get("/users/:user_id", users.GetUser)
//
//	//post routes
//	app.Post("/food", middleware.AuthMiddleware(), foods.SaveFood)
//	app.Put("/food/:food_id", middleware.AuthMiddleware(), foods.UpdateFood)
//	app.Get("/food/:food_id", foods.GetFoodAndCreator)
//	app.Delete("/food/:food_id", middleware.AuthMiddleware(), foods.DeleteFood)
//	app.Get("/food", foods.GetAllFood)
//
//	//authentication routes
//	app.Post("/login", authenticate.Login)
//	app.Post("/logout", authenticate.Logout)
//	app.Post("/refresh", authenticate.Refresh)
//}
