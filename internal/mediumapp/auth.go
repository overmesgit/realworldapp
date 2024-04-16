package mediumapp

import (
	"crypto/sha256"
	"errors"
	"example.com/medium/ent"
	"example.com/medium/ent/user"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"net/http"
	"time"
)

var AuthSecret = []byte("secretKey")

type UserContext struct {
	ID       int
	UserName string
	SignedIn bool
}

func InitAuthController(e *echo.Echo, client *ent.Client) {
	controller := AuthController{client}
	e.GET("/sign-up", func(c echo.Context) error {
		errMap := make(map[string][]string)
		return render(c, http.StatusOK, signUp(errMap, UserContext{}))
	})
	e.POST("/sign-up", controller.SingUp)
	e.GET("/sign-in", func(c echo.Context) error {
		errMap := make(map[string][]string)
		return render(c, http.StatusOK, signIn(errMap, UserContext{}))
	})
	e.POST("/sign-in", controller.SingIn)
}

type AuthController struct {
	*ent.Client
}

type jwtCustomClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var jwtTokenMissingOrInvalid = errors.New("JWT token missing or invalid")

func getJwtClaims(c echo.Context) (*jwtCustomClaims, error) {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return nil, jwtTokenMissingOrInvalid
	}
	claims, ok := token.Claims.(*jwtCustomClaims) // by default claims is of type `jwt.MapClaims`
	if !ok {
		return nil, errors.New("failed to cast claims as jwt.MapClaims")
	}
	return claims, nil
}

func GetUserContext(c echo.Context) (UserContext, error) {
	claim, err := getJwtClaims(c)
	if err != nil {
		return UserContext{}, err
	}
	return UserContext{ID: claim.UserID, UserName: claim.Username, SignedIn: true}, nil
}

type User struct {
	UserName string `form:"username" validate:"required,gte=5"`
	Password string `form:"password" validate:"required,gte=5"`
}

func HashPassword(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

func (controller AuthController) SingUp(c echo.Context) error {
	user := new(User)
	if err := c.Bind(user); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	errMap := GetErrorMap(validate.Struct(user))
	if errMap != nil {
		return render(c, http.StatusBadRequest, signUp(errMap, UserContext{}))
	}
	userObj, saveErr := controller.Client.User.
		Create().
		SetName(user.UserName).
		SetPassword(HashPassword(user.Password)).
		Save(c.Request().Context())
	var pgErr *pq.Error
	if errors.As(saveErr, &pgErr) {
		if pgErr.Code == "23505" {
			errMap := map[string][]string{
				"username": {"User with such name already exists."},
			}
			return render(c, http.StatusBadRequest, signUp(errMap, UserContext{}))
		}
	}
	if saveErr != nil {
		return saveErr
	}

	tokenString, tokenErr := controller.createJWTToken(userObj.ID, userObj.Name)
	if tokenErr != nil {
		return tokenErr
	}

	c.SetCookie(&http.Cookie{Name: "user", Value: tokenString})
	return c.Redirect(http.StatusSeeOther, "/")
}

func (controller AuthController) createJWTToken(userID int, userName string) (string, error) {
	claims := &jwtCustomClaims{
		userID,
		userName,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(AuthSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (controller AuthController) SingIn(c echo.Context) error {
	userForm := new(User)
	if err := c.Bind(userForm); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	errMap := GetErrorMap(validate.Struct(userForm))
	if errMap != nil {
		return render(c, http.StatusBadRequest, signIn(errMap, UserContext{}))
	}
	userObj, queryErr := controller.Client.User.
		Query().
		Where(user.Name(userForm.UserName)).
		Where(user.Password(HashPassword(userForm.Password))).
		Only(c.Request().Context())
	var notFound *ent.NotFoundError
	if errors.As(queryErr, &notFound) {
		errMap := map[string][]string{
			"username": {"User with such name and password doesn't exist."},
		}
		return render(c, http.StatusBadRequest, signIn(errMap, UserContext{}))
	}
	if queryErr != nil {
		return queryErr
	}

	tokenString, tokenErr := controller.createJWTToken(userObj.ID, userObj.Name)
	if tokenErr != nil {
		return tokenErr
	}

	c.SetCookie(&http.Cookie{Name: "user", Value: tokenString})
	return c.Redirect(http.StatusSeeOther, "/")
}
