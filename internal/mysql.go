package internal

import (
	"errors"
	"fmt"
	"io"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func InitDB() {
	config := GlobalConfig.MySQL
	
	newLogger := logger.New(
		log.New(io.Discard, "\r\n", log.LstdFlags), 
		logger.Config{
			SlowThreshold:             config.SlowThreshold,
			LogLevel:                  logger.LogLevel(config.LogLevel),
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	)
	
	var err error
	db, err = gorm.Open(mysql.Open(config.DSN), &gorm.Config{
		Logger: newLogger,
	})
	
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}
	
	initTable()
	fmt.Println("Database initialized")
}

// MySQL has two tables: course and order
type Course struct {
	ID uint `gorm:"primaryKey"`
	Title string `gorm:"not null"`
	Hotpoint bool `gorm:"default:false"`
	Description string `gorm:"type:text"`
	Stock int `gorm:"default:0"`
}

type Order struct {
	ID uint `gorm:"primaryKey"`
	UserID int `gorm:"not null"`
	CourseID int `gorm:"not null"`
	Status string `gorm:"default:pending"`
}

func initTable() {
	db.Migrator().DropTable(&Course{})
	db.Migrator().DropTable(&Order{})
	db.AutoMigrate(&Course{})
	db.AutoMigrate(&Order{})
	newCourseList := []Course{
		{
			Title: "CSC3050",
			Hotpoint: false,
			Description: "This course is about computer architecture.",
			Stock: 150,
		},
		{
			Title: "CSC3100",
			Hotpoint: false,
			Description: "This course is about computer data structure.",
			Stock: 100,
		},
		{
			Title: "CSC3150",
			Hotpoint: true,
			Description: "This course is about computer operating system.",
			Stock: 150,
		},
		{
			Title: "CSC3170",
			Hotpoint: true,
			Description: "This course is about computer database.",
			Stock: 150,
		},
		{
			Title: "CSC4001",
			Hotpoint: false,
			Description: "This course is about software engineering(most about testing).",
			Stock: 100,
		},
		{
			Title: "CSC4120",
			Hotpoint: true,
			Description: "This course is about advance algorithm.",
			Stock: 200,
		},
	}
	db.Create(&newCourseList)
}

func getAllCourses() []Course {
	var courses []Course
	db.Find(&courses)
	return courses
}

func getAllOrders() []Order {
	var orders []Order
	db.Find(&orders)
	return orders
}

// core function of the final step
// check the replication and stock, then create order
func createOrder(uid int, cid int) error {
	rollback := func(tx *gorm.DB, msg string) error {
		tx.Rollback()
		rollbackRedis(cid)
		if msg != "replicated order" {
			changeOrderStatus(uid, cid, -1)
		}
		return errors.New(msg)
	}
	var err error
	var order Order

	// begin a transaction, all operations are atomic
	tx := db.Begin()

	// check if the order already exists
	result := tx.Model(&Order{}).Where("user_id = ? and course_id = ?", uid, cid).First(&order)
	if result.Error == nil {
		return rollback(tx, "replicated order")
	}
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// continue excute tx
	} else {
		return rollback(tx, "query order failed")
	}

	// decrease the stock of the course if enough stock
	if err = tx.Model(&Course{}).Where("id = ?", cid).Update("stock", gorm.Expr("stock - 1")).Error; err != nil {
		return rollback(tx, "update stock failed")
	}

	// create order
	if err = tx.Create(&Order{UserID: uid, CourseID: cid}).Error; err != nil {
		return rollback(tx, "create order failed")
	}

	tx.Commit()
	// change the status of the order in the redis cache for front end polling
	changeOrderStatus(uid, cid, 1)
	fmt.Printf("create order success -> %d:%d\n", uid, cid)
	return nil
}

// print the length of stock table and order table
func PrintStockAndOrder() {
	var courses []Course
	db.Find(&courses)
	fmt.Println("stock table:")
	for _, course := range courses {
		fmt.Printf("id: %d, stock: %d\n", course.ID, course.Stock)
	}
	var orders []Order
	db.Find(&orders)
	fmt.Println("order table:")
	fmt.Printf("order table length: %d\n", len(orders))
}