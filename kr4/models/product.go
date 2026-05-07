package models

// Product — сущность для Task 9.1 (миграции БД)
type Product struct {
	ID          uint    `json:"id"          gorm:"column:id;primaryKey;autoIncrement"`
	Title       string  `json:"title"       gorm:"column:title;not null"`
	Price       float64 `json:"price"       gorm:"column:price;not null"`
	Count       int     `json:"count"       gorm:"column:count;not null"`
	Description string  `json:"description" gorm:"column:description;not null;default:''"`
}

type ProductInput struct {
	Title       string  `json:"title"       binding:"required"`
	Price       float64 `json:"price"       binding:"required,gt=0"`
	Count       int     `json:"count"       binding:"gte=0"`
	Description string  `json:"description"`
}
