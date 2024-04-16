package mediumapp

import (
	"errors"
	"example.com/medium/ent"
	"example.com/medium/ent/article"
	"github.com/a-h/templ"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func render(ctx echo.Context, status int, t templ.Component) error {
	ctx.Response().Writer.WriteHeader(status)

	err := t.Render(ctx.Request().Context(), ctx.Response().Writer)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "failed to render response template")
	}

	return nil
}

func StartServer(client *ent.Client) {
	insecure := echo.New()
	insecure.Use(middleware.Logger())
	insecure.Static("/static", "assets")

	InitAuthController(insecure, client)
	secure := insecure.Group("")
	config := echojwt.Config{
		ContinueOnIgnoredError: true,
		ErrorHandler: func(c echo.Context, err error) error {
			return nil
		},
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(jwtCustomClaims)
		},
		SigningKey:  AuthSecret,
		TokenLookup: "cookie:user",
	}
	secure.Use(echojwt.WithConfig(config))

	secure.GET("/", func(c echo.Context) error {
		userContext, err := GetUserContext(c)
		if err != nil && !errors.As(err, &jwtTokenMissingOrInvalid) {
			return err
		}
		articles, err := client.Article.
			Query().WithUser().Order(ent.Desc(article.FieldID)).
			All(c.Request().Context())
		if err != nil {
			return err
		}
		return index(articles, userContext).Render(c.Request().Context(), c.Response().Writer)
	})

	secure.GET("/user", func(c echo.Context) error {
		user, err := getJwtClaims(c)

		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, user)
	})

	InitArticleController(secure, client)
	insecure.Logger.Fatal(insecure.Start(":8080"))
}
