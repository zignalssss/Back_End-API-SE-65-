package main

import (
	"fmt"
	"go-server/adapters"
	"go-server/core"

	_ "go-server/docs"
	"go-server/middleware"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"gopkg.in/gomail.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// @title TiawPao API Documentation
// @version 1.0
// @description API documentation สำหรับ Fiber + Swagger
// @host localhost:8000
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
const (
	host     = "localhost"  // or the Docker service name if running in another container
	port     = 5432         // default PostgreSQL port
	user     = "myuser"     // as defined in docker-compose.yml
	password = "mypassword" // as defined in docker-compose.yml
	dbname   = "mydatabase" // as defined in docker-compose.yml
)

func main() {

	app := fiber.New()
	// Configure your PostgreSQL database details here

	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins: "*", // Change this to your frontend domain for security
	// 	AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	// 	AllowHeaders: "Origin, Content-Type, Accept",
	// }))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // Change this to specific frontend domains for better security
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	dsn := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// New logger for detailed SQL logging
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second, // Slow SQL threshold
			LogLevel:      logger.Info, // Log level
			Colorful:      true,        // Enable color
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger, // add Logger
	})

	if err != nil {
		panic("failed to connect to database")
	}

	env_err := godotenv.Load()
	if env_err != nil {
		// log.Fatal("Error loading .env file")
		panic("Error loading .env file")
	}

	smtpHost := os.Getenv("MAILER_HOST")
	smtpPort, _ := strconv.Atoi(os.Getenv("MAILER_PORT")) // Convert port to int
	smtpUser := os.Getenv("MAILER_USERNAME")
	smtpPass := os.Getenv("MAILER_PASSWORD")

	dialer := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)
	middleware.InitFirebase()
	//Implement Port Hexagonal Arc {Secondary to Primary Port}
	userRepo := adapters.NewGormUserRepository(db)
	userService := core.NewUserService(userRepo)
	// userHandler := adapters.NewHttpUserHandler(userService)
	emailRepo := adapters.NewEmailRepository(dialer)
	emailService := core.NewEmailService(emailRepo)
	// Assuming core.NewEmailService(emailRepo) creates an email service
	userHandler := adapters.NewHttpUserHandler(userService, emailService)
	// Swagger UI Route
	app.Static("/access/images", "./access/images")
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// API Routes
	api := app.Group("/api/v1")
	api.Post("/user/register", userHandler.RegisterUser)
	api.Get("/user/getuser/:email", middleware.AuthMiddleware, userHandler.GetUser)
	// api.Get("/user/getuser/:email", middleware.JWTProtected(), userHandler.GetUser)
	api.Get("/user/genotp/:email", userHandler.GenOTP)
	api.Post("/user/verifyotp", userHandler.VerifyOTP)
	api.Post("/user/createplan", middleware.AuthMiddleware, userHandler.CreatePlanTrip)
	api.Put("/user/update/:email", middleware.AuthMiddleware, userHandler.UserUpdate)
	api.Post("/user/uploadImage", userHandler.UploadImage)
	api.Put("/user/updateuserplan/:email", middleware.AuthMiddleware, userHandler.UserUpdatePlanByEmail)
	api.Delete("/user/deleteuserplanbyemail/:email", middleware.AuthMiddleware, userHandler.DeleteUserPlanByEmailHandler)
	api.Get("/plan/gettriplocation/:id", userHandler.GetTripLocationHandler)
	api.Get("/plan/getplanbyid/:id", userHandler.GetPlanByIDHandler)
	api.Get("/plan/getpublicplan", userHandler.GetVisiblePlansHandler)
	api.Post("/plan/addtriplocation/:id", middleware.AuthMiddleware, userHandler.AddTripLocationHandler)
	api.Put("/plan/updateplan/:id", middleware.AuthMiddleware, userHandler.UpdatePlanByID)
	api.Put("/plan/updateauthorimg/:id", middleware.AuthMiddleware, userHandler.UpdateAuthorImgHandler)
	api.Put("/plan/updateauthorname/:id", middleware.AuthMiddleware, userHandler.UpdateAuthorNameHandler)
	api.Delete("/plan/deleteplanbyid/:id", middleware.AuthMiddleware, userHandler.DeletePlanByIDHandler)
	api.Delete("/plan/deletetriplocation/:id", middleware.AuthMiddleware, userHandler.DeleteTripLocationHandler)
	api.Post("/admin/register", userHandler.RegisterAdmin)
	api.Post("/admin/login", userHandler.LoginAdmin)
	api.Get("/admin/getAllUsers", middleware.JWTProtected(), userHandler.GetAllUsers)
	api.Get("/admin/getAllPlans", middleware.JWTProtected(), userHandler.GetAllPlans)
	// Migrate the schema
	db.AutoMigrate(&core.User{})
	db.AutoMigrate(&core.Admin{})
	db.AutoMigrate(&core.Verification{})
	db.AutoMigrate(&core.Plan{})

	fmt.Println("Database migration completed!")
	app.Listen(("0.0.0.0:8000"))
	// newBook := &Book{Name: "Think Again", Author: "adam", Description: "test", price: 200}

	// createBook(db, newBook)
}
