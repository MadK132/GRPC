package inventory

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Product struct {
	ID       string  `json:"id" bson:"_id"`
	Name     string  `json:"name" bson:"name"`
	Price    float64 `json:"price" bson:"price"`
	Stock    int     `json:"stock" bson:"stock"`
	Category string  `json:"category" bson:"category"`
}

type Discount struct {
	ID                string    `json:"id" bson:"_id"`
	Name              string    `json:"name" bson:"name"`
	Description       string    `json:"description" bson:"description"`
	DiscountPercentage float64  `json:"discount_percentage" bson:"discount_percentage"`
	ApplicableProducts []string `json:"applicable_products" bson:"applicable_products"`
	StartDate         time.Time `json:"start_date" bson:"start_date"`
	EndDate           time.Time `json:"end_date" bson:"end_date"`
	IsActive          bool      `json:"is_active" bson:"is_active"`
}

var client *mongo.Client
var collection *mongo.Collection
var discountCollection *mongo.Collection

func StartService() {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}

	collection = client.Database("ecommerce").Collection("products")
	discountCollection = client.Database("ecommerce").Collection("discounts")

	r := gin.Default()

	r.POST("/products", createProduct)
	r.GET("/products/:id", getProduct)
	r.PATCH("/products/:id", updateProduct)
	r.DELETE("/products/:id", deleteProduct)
	r.GET("/products", listProducts)

	// Add new discount endpoints
	r.POST("/discounts", createDiscount)
	r.GET("/products/promotions", getAllProductsWithPromotion)
	r.DELETE("/discounts/:id", deleteDiscount)

	r.Run(":8081")
}

func createProduct(c *gin.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

func getProduct(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var product Product
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

func updateProduct(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var updateData Product
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := bson.M{}
	if updateData.Name != "" {
		update["name"] = updateData.Name
	}
	if updateData.Price != 0 {
		update["price"] = updateData.Price
	}
	if updateData.Stock != 0 {
		update["stock"] = updateData.Stock
	}
	if updateData.Category != "" {
		update["category"] = updateData.Category
	}

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	var updatedProduct Product
	err = collection.FindOne(ctx, bson.M{"_id": id}).Decode(&updatedProduct)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedProduct)
}

func deleteProduct(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}

func listProducts(c *gin.Context) {
	// Filter parameters
	category := c.Query("category")
	minPrice := c.Query("min_price")
	maxPrice := c.Query("max_price")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	filter := bson.M{}
	if category != "" {
		filter["category"] = category
	}
	if minPrice != "" {
		min, err := strconv.ParseFloat(minPrice, 64)
		if err == nil {
			filter["price"] = bson.M{"$gte": min}
		}
	}
	if maxPrice != "" {
		max, err := strconv.ParseFloat(maxPrice, 64)
		if err == nil {
			if _, exists := filter["price"]; exists {
				filter["price"].(bson.M)["$lte"] = max
			} else {
				filter["price"] = bson.M{"$lte": max}
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	totalItems, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := (totalItems + int64(limit) - 1) / int64(limit)
	skip := int64((page - 1) * limit)

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(skip)

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var products []Product
	if err = cursor.All(ctx, &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"metadata": gin.H{
			"total_items":  totalItems,
			"total_pages":  totalPages,
			"current_page": page,
			"limit":        limit,
		},
	})
}

func createDiscount(c *gin.Context) {
	var discount Discount
	if err := c.ShouldBindJSON(&discount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := discountCollection.InsertOne(ctx, discount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, discount)
}

func getAllProductsWithPromotion(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get active discounts
	currentTime := time.Now()
	filter := bson.M{
		"is_active": true,
		"start_date": bson.M{"$lte": currentTime},
		"end_date": bson.M{"$gte": currentTime},
	}

	cursor, err := discountCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var discounts []Discount
	if err = cursor.All(ctx, &discounts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get all applicable product IDs
	productIDs := make(map[string]struct{})
	for _, discount := range discounts {
		for _, productID := range discount.ApplicableProducts {
			productIDs[productID] = struct{}{}
		}
	}

	// Convert map keys to slice
	ids := make([]string, 0, len(productIDs))
	for id := range productIDs {
		ids = append(ids, id)
	}

	// Get products with active discounts
	productsFilter := bson.M{"_id": bson.M{"$in": ids}}
	productsCursor, err := collection.Find(ctx, productsFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer productsCursor.Close(ctx)

	var products []Product
	if err = productsCursor.All(ctx, &products); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create response with products and their applicable discounts
	type ProductWithDiscount struct {
		Product   Product    `json:"product"`
		Discounts []Discount `json:"discounts"`
	}

	productsWithDiscounts := make([]ProductWithDiscount, 0)
	for _, product := range products {
		applicable := make([]Discount, 0)
		for _, discount := range discounts {
			for _, pid := range discount.ApplicableProducts {
				if pid == product.ID {
					applicable = append(applicable, discount)
					break
				}
			}
		}
		productsWithDiscounts = append(productsWithDiscounts, ProductWithDiscount{
			Product:   product,
			Discounts: applicable,
		})
	}

	c.JSON(http.StatusOK, productsWithDiscounts)
}

func deleteDiscount(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := discountCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Discount not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Discount deleted"})
}
