package main

import (
	"encoding/base32"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/banzaicloud/bank-vaults/auth"
	"github.com/banzaicloud/bank-vaults/database"
	"github.com/banzaicloud/bank-vaults/vault"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func vaultExample() {
	client, err := vault.NewClient("default")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Created Vault client.")

	secret, err := client.Vault().Logical().List("secret/accesstokens")
	if err != nil {
		log.Fatal(err)
	}
	if secret != nil {
		for _, v := range secret.Data {
			log.Printf("-> %+v", v)
			for _, v := range v.([]interface{}) {
				log.Printf("-> %+v", v)
			}
		}

	} else {
		log.Println("Found nothing")
		time.Sleep(time.Minute)
		client.Close()
		time.Sleep(time.Hour)
	}
}

func gormExample() {
	secretSource, err := database.DynamicSecretDataSource("mysql", "my-role@tcp(127.0.0.1:3306)/sparky?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("use this with GORM:\ndb, err := gorm.Open(\"mysql\", \"%s\")", secretSource)
}

func authExample() {
	userID := "1"
	tokenID := "123"
	signingKey := "mys3cr3t"
	signingKeyBase32 := base32.StdEncoding.EncodeToString([]byte(signingKey))

	// Issue a JWT token for the end user
	claims := &auth.ScopedClaims{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  jwt.TimeFunc().Unix(),
			ExpiresAt: 0,
			Subject:   userID,
			Id:        tokenID,
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := jwtToken.SignedString([]byte(signingKeyBase32))
	if err != nil {
		panic(err)
	}

	// Use this curl command
	fmt.Printf("curl -H \"Authorization: Bearer %s\" -v http://localhost:9091/\n\n", signedToken)

	// Create a Vault Token store and insert the mirroring token
	tokenStore := auth.NewVaultTokenStore("")
	err = tokenStore.Store(userID, auth.NewToken(tokenID, "my test token"))
	if err != nil {
		panic(err)
	}

	// In the protected application you only need this part:
	// Start a Gin engine, serving and API protected via the JWT Middleware
	engine := gin.New()
	engine.Use(auth.JWTAuth(tokenStore, signingKey, nil))
	engine.Use(gin.Logger(), gin.ErrorLogger())
	engine.GET("/", func(c *gin.Context) {
		user := c.Request.Context().Value(auth.CurrentUser)
		c.JSON(200, gin.H{"claims": user})
	})
	engine.Run(":9091")
}

// REQUIRED to start a Vault 0.9 dev server with:
// vault server -dev &
func main() {
	os.Setenv("VAULT_ADDR", "http://localhost:8200")
	// vaultExample()
	// gormExample()
	authExample()
}
